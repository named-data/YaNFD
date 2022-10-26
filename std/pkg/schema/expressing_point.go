package schema

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type ExpressPoint struct {
	BaseNode

	onInt           *Event[*NodeOnIntEvent]
	onValidateInt   *Event[*NodeValidateEvent]
	onValidateData  *Event[*NodeValidateEvent]
	onSearchStorage *Event[*NodeSearchStorageEvent]

	intSigner   ndn.Signer
	canBePrefix bool
	mustBeFresh bool
	lifetime    time.Duration
	supressInt  bool
}

type intResult struct {
	result     ndn.InterestResult
	data       ndn.Data
	rawData    enc.Wire
	sigCovered enc.Wire
	nackReason uint64
}

func (n *ExpressPoint) SearchCache(matching enc.Matching, name enc.Name, context Context) enc.Wire {
	cachedData := enc.Wire(nil)
	ok := true
	context[CkCachedData] = cachedData
	for _, evt := range n.onSearchStorage.val {
		(*evt)(matching, name, context)
		cachedData, ok = context[CkCachedData].(enc.Wire)
		if ok && len(cachedData) > 0 {
			return cachedData
		}
	}
	return nil
}

// OnInterest is the function called when an Interest comes.
func (n *ExpressPoint) OnInterest(
	interest ndn.Interest, rawInterest enc.Wire, sigCovered enc.Wire,
	reply ndn.ReplyFunc, deadline time.Time, matching enc.Matching,
) {
	context := Context{
		CkInterest:   interest,
		CkDeadline:   deadline,
		CkRawPacket:  rawInterest,
		CkSigCovered: sigCovered,
		CkName:       interest.Name(),
		CkEngine:     n.engine,
		CkContent:    interest.AppParam(),
	}

	// Search storage
	// Reply if there is data (including AppNack). No further callback will be called if hit.
	// This is the same behavior as a forwarder.
	cachedData := n.SearchCache(matching, interest.Name(), context)
	if len(cachedData) > 0 {
		err := reply(cachedData)
		if err != nil {
			n.Log.WithField("name", interest.Name().String()).Error("Unable to reply Interest. Drop.")
		}
		return
	}

	go func() {
		// Validate Interest
		// Only done when there is a signature.
		if interest.Signature().SigType() != ndn.SignatureNone || interest.AppParam() != nil {
			validRes := VrSilence
			context[CkLastValidResult] = validRes
			for _, evt := range n.onValidateInt.val {
				res := (*evt)(matching, interest.Name(), interest.Signature(), sigCovered, context)
				if res < VrSilence {
					n.Log.WithField("name", interest.Name().String()).Warn("Verification failed for Interest. Drop.")
					return
				}
				validRes = utils.Max(validRes, res)
				context[CkLastValidResult] = validRes
				if validRes >= VrBypass {
					break
				}
			}
			if validRes <= VrSilence {
				n.Log.WithField("name", interest.Name().String()).Warn("Unverified Interest. Drop.")
				return
			}
		}

		// PreRecvInt
		// Used to decrypt AppParam or handle before onInterest hits, if applicable.
		// Do we need them? Hold for now.

		// OnInt
		isDone := false
		for _, evt := range n.onInt.val {
			isDone = (*evt)(matching, interest.AppParam(), reply, context)
			if isDone {
				break
			}
		}

		// PreSendData
		// Used to encrypt Data or handle after onInterest hits, if applicable.
		// Do we need them? Hold for now.
	}()
}

// Need is the function to obtain the corresponding Data. May express an Interest if the Data is not stored.
func (n *ExpressPoint) Need(
	matching enc.Matching, name enc.Name, appParam enc.Wire, context Context,
) (ndn.InterestResult, enc.Wire) {
	// Construct the name (excluding hash) if not yet
	if name == nil {
		name = n.Apply(matching)
		if name == nil {
			n.Log.Error("Unable to construct Interest Name in Need().")
			return ndn.InterestResultNone, nil
		}
	}

	// If appParam is empty and not signed, the Interest name is final.
	// Otherwise, we have to construct the Interest first before searching storage.
	context[CkEngine] = n.engine
	signer := n.intSigner
	// storageSearched := false
	if signer == nil && appParam == nil {
		cachedData := n.SearchCache(matching, name, context)
		if cachedData != nil {
			return ndn.InterestResultData, cachedData
		}
		// storageSearched = true
	}

	// Construst Interest
	engine := n.engine
	spec := engine.Spec()
	timer := engine.Timer()
	intCfg := ndn.InterestConfig{
		CanBePrefix:    n.canBePrefix,
		MustBeFresh:    n.mustBeFresh,
		Lifetime:       utils.IdPtr(n.lifetime),
		Nonce:          utils.ConvertNonce(timer.Nonce()),
		HopLimit:       nil,
		ForwardingHint: nil,
	}
	if ctxVal, ok := context[CkCanBePrefix]; ok {
		if v, ok := ctxVal.(bool); ok {
			intCfg.CanBePrefix = v
		}
	}
	if ctxVal, ok := context[CkMustBeFresh]; ok {
		if v, ok := ctxVal.(bool); ok {
			intCfg.MustBeFresh = v
		}
	}
	if ctxVal, ok := context[CkLifetime]; ok {
		if v, ok := ctxVal.(time.Duration); ok {
			intCfg.Lifetime = &v
		}
	}
	if ctxVal, ok := context[CkNonce]; ok {
		if v, ok := ctxVal.(uint64); ok {
			intCfg.Nonce = &v
		}
	}
	if ctxVal, ok := context[CkHopLimit]; ok {
		if v, ok := ctxVal.(uint); ok {
			intCfg.HopLimit = utils.IdPtr(utils.Min(v, 255))
		}
	}
	if ctxVal, ok := context[CkForwardingHint]; ok {
		if v, ok := ctxVal.([]enc.Name); ok {
			intCfg.ForwardingHint = v
		}
	}
	wire, _, finalName, err := spec.MakeInterest(name, &intCfg, appParam, signer)
	if err != nil {
		n.Log.Errorf("Unable to encode Interest in Need(): %+v", err)
		return ndn.InterestResultNone, nil
	}

	// We may search the storage if not yet
	// if !storageSearched {
	// 	// Since it is not useful generally, skip for now.
	// }
	if n.supressInt {
		return ndn.InterestResultNack, nil
	}

	// Set the deadline
	// assert(intCfg.Lifetime != nil)
	if _, ok := context[CkDeadline]; !ok {
		deadline := timer.Now().Add(*intCfg.Lifetime)
		context[CkDeadline] = deadline
	}

	// Express the Interest
	// Note that this function runs on a different go routine than the callback.
	ch := make(chan intResult)
	err = engine.Express(finalName, &intCfg, wire,
		func(result ndn.InterestResult, data ndn.Data, rawData, sigCovered enc.Wire, nackReason uint64) {
			ch <- intResult{
				result:     result,
				data:       data,
				rawData:    rawData,
				sigCovered: sigCovered,
				nackReason: nackReason,
			}
			close(ch)
		})
	if err != nil {
		n.Log.WithField("name", finalName.String()).Warn("Failed to express Interest.")
		return ndn.InterestResultNone, nil
	}
	intRet := <-ch
	if intRet.result != ndn.InterestResultData {
		if intRet.result == ndn.InterestResultNack {
			context[CkNackReason] = intRet.nackReason
		}
		return intRet.result, nil
	}
	data := intRet.data
	context[CkName] = data.Name()
	context[CkData] = data
	context[CkRawPacket] = intRet.rawData
	context[CkSigCovered] = intRet.sigCovered
	context[CkContent] = data.Content()

	// Validate data
	validRes := VrSilence
	context[CkLastValidResult] = validRes
	for _, evt := range n.onValidateData.val {
		res := (*evt)(matching, data.Name(), data.Signature(), intRet.sigCovered, context)
		if res < VrSilence {
			n.Log.WithField("name", data.Name().String()).Warn("Verification failed for Data. Drop.")
			context[CkLastValidResult] = res
			return ndn.InterestResultUnverified, nil
		}
		validRes = utils.Max(validRes, res)
		context[CkLastValidResult] = validRes
		if validRes >= VrBypass {
			break
		}
	}
	if validRes <= VrSilence {
		n.Log.WithField("name", data.Name().String()).Warn("Unverified Data. Drop.")
		return ndn.InterestResultUnverified, nil
	}

	// Return the result
	return ndn.InterestResultData, data.Content()
}

// Get a property or callback event
func (n *ExpressPoint) Get(propName PropKey) any {
	if ret := n.BaseNode.Get(propName); ret != nil {
		return ret
	}
	switch propName {
	case PropOnInterest:
		return n.onInt
	case PropOnValidateInt:
		return n.onValidateInt
	case PropOnValidateData:
		return n.onValidateData
	case PropOnSearchStorage:
		return n.onSearchStorage
	case PropCanBePrefix:
		return n.canBePrefix
	case PropMustBeFresh:
		return n.mustBeFresh
	case PropLifetime:
		return n.lifetime
	case PropIntSigner:
		return n.intSigner
	case PropSuppressInt:
		return n.supressInt
	}
	return nil
}

// Set a property. Use Get() to update callback events.
func (n *ExpressPoint) Set(propName PropKey, value any) error {
	if ret := n.BaseNode.Set(propName, value); ret == nil {
		return ret
	}
	switch propName {
	case PropCanBePrefix:
		return propertySet(&n.canBePrefix, propName, value)
	case PropMustBeFresh:
		return propertySet(&n.mustBeFresh, propName, value)
	case PropLifetime:
		return propertySet(&n.lifetime, propName, value)
	case PropIntSigner:
		return propertySet(&n.intSigner, propName, value)
	case PropSuppressInt:
		return propertySet(&n.supressInt, propName, value)
	}
	return ndn.ErrNotSupported{Item: string(propName)}
}

func (n *ExpressPoint) Init(parent NTNode, edge enc.ComponentPattern) {
	n.BaseNode.Init(parent, edge)
	n.onInt = NewEvent[*NodeOnIntEvent]()
	n.onValidateInt = NewEvent[*NodeValidateEvent]()
	n.onValidateData = NewEvent[*NodeValidateEvent]()
	n.onSearchStorage = NewEvent[*NodeSearchStorageEvent]()

	n.intSigner = nil
	n.canBePrefix = false
	n.mustBeFresh = true
	n.lifetime = 4 * time.Second
	n.supressInt = false

	n.Self = n
}
