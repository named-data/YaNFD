package schema

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// ExpressPoint is an expressing point where an Interest is supposed to be expressed.
// For example, for an RDR protocol w/ metadata name being /prefix/<filename>/32=metadata/<v=version>,
// The node at "/prefix/<filename>/32=metadata" is an ExpressPoint,
// as the consumer will express an Interest of this name with CanBePrefix.
// The node at "/prefix/<filename>/32=metadata/<v=version>" is a LeafNode.
type ExpressPoint struct {
	BaseNode

	onInt           *Event[*NodeOnIntEvent]
	onValidateInt   *Event[*NodeValidateEvent]
	onValidateData  *Event[*NodeValidateEvent]
	onSearchStorage *Event[*NodeSearchStorageEvent]
	onSaveStorage   *Event[*NodeSaveStorageEvent]

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

func (n *ExpressPoint) SearchCache(
	matching enc.Matching, name enc.Name, canBePrefix bool, mustBeFresh bool, context Context,
) enc.Wire {
	cachedData := enc.Wire(nil)
	for _, evt := range n.onSearchStorage.val {
		cachedData = (*evt)(matching, name, canBePrefix, mustBeFresh, context)
		if len(cachedData) > 0 {
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
	cachedData := n.SearchCache(matching, interest.Name(), interest.CanBePrefix(), interest.MustBeFresh(), context)
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
		// TODO: Validate Sha256 in name
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
// Name is constructed from matching if nil. If given, name must agree with matching.
// TODO:
// 1. Need we make it non-blocking and return future/channel?
// 2. Need we use a different type than ndn.InterestResult?
// 3. Need we use []byte instead of enc.Wire, given that the performance is not a big consideration here?
func (n *ExpressPoint) Need(
	matching enc.Matching, name enc.Name, appParam enc.Wire, context Context,
) chan NeedResult {
	ret := make(chan NeedResult, 1)
	// Construct the name (excluding hash) if not yet
	if name == nil {
		name = n.Apply(matching)
		if name == nil {
			n.Log.Error("Unable to construct Interest Name in Need().")
			ret <- NeedResult{ndn.InterestResultNone, nil}
			close(ret)
			return ret
		}
	}

	// If appParam is empty and not signed, the Interest name is final.
	// Otherwise, we have to construct the Interest first before searching storage.
	engine := n.engine
	spec := engine.Spec()
	timer := engine.Timer()
	context[CkEngine] = engine
	signer := n.intSigner
	// storageSearched := false
	canBePrefix := n.canBePrefix
	mustBeFresh := n.mustBeFresh
	if ctxVal, ok := context[CkCanBePrefix]; ok {
		if v, ok := ctxVal.(bool); ok {
			canBePrefix = v
		}
	}
	if ctxVal, ok := context[CkMustBeFresh]; ok {
		if v, ok := ctxVal.(bool); ok {
			mustBeFresh = v
		}
	}
	if signer == nil && appParam == nil {
		cachedData := n.SearchCache(matching, name, canBePrefix, mustBeFresh, context)
		if cachedData != nil {
			data, sigCovered, err := spec.ReadData(enc.NewWireReader(cachedData))
			if err == nil {
				context[CkName] = data.Name()
				context[CkData] = data
				context[CkRawPacket] = cachedData
				context[CkSigCovered] = sigCovered
				context[CkContent] = data.Content()
				context[CkLastValidResult] = VrCachedData
				ret <- NeedResult{ndn.InterestResultData, data.Content()}
				close(ret)
				return ret
			} else {
				n.Log.WithField("name", name.String()).Error("The storage returned an invalid data")
			}
		}
		// storageSearched = true
	}

	// Construst Interest
	intCfg := ndn.InterestConfig{
		CanBePrefix:    canBePrefix,
		MustBeFresh:    mustBeFresh,
		Lifetime:       utils.IdPtr(n.lifetime),
		Nonce:          utils.ConvertNonce(timer.Nonce()),
		HopLimit:       nil,
		ForwardingHint: nil,
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
		n.Log.WithField("name", name.String()).Errorf("Unable to encode Interest in Need(): %+v", err)
		ret <- NeedResult{ndn.InterestResultNone, nil}
		close(ret)
		return ret
	}

	// We may search the storage if not yet
	// if !storageSearched {
	// 	// Since it is not useful generally, skip for now.
	// }
	if n.supressInt {
		ret <- NeedResult{ndn.InterestResultNack, nil}
		close(ret)
		return ret
	}
	if supress, ok := context[CkSupressInt].(bool); ok && supress {
		ret <- NeedResult{ndn.InterestResultNack, nil}
		close(ret)
		return ret
	}

	// Set the deadline
	// assert(intCfg.Lifetime != nil)
	if _, ok := context[CkDeadline]; !ok {
		deadline := timer.Now().Add(*intCfg.Lifetime)
		context[CkDeadline] = deadline
	}

	// Express the Interest
	// Note that this function runs on a different go routine than the callback.
	// To avoid clogging the engine, the callback needs to return ASAP, so an inner goroutine is created.
	err = engine.Express(finalName, &intCfg, wire,
		func(result ndn.InterestResult, data ndn.Data, rawData, sigCovered enc.Wire, nackReason uint64) {
			if result != ndn.InterestResultData {
				if result == ndn.InterestResultNack {
					context[CkNackReason] = nackReason
				}
				ret <- NeedResult{result, nil}
				close(ret)
				return
			}

			go func() {
				context[CkName] = data.Name()
				context[CkData] = data
				context[CkRawPacket] = rawData
				context[CkSigCovered] = sigCovered
				context[CkContent] = data.Content()

				// Validate data
				validRes := VrSilence
				context[CkLastValidResult] = validRes
				for _, evt := range n.onValidateData.val {
					res := (*evt)(matching, data.Name(), data.Signature(), sigCovered, context)
					if res < VrSilence {
						n.Log.WithField("name", data.Name().String()).Warn("Verification failed for Data. Drop.")
						context[CkLastValidResult] = res
						ret <- NeedResult{ndn.InterestResultUnverified, nil}
						close(ret)
						return
					}
					validRes = utils.Max(validRes, res)
					context[CkLastValidResult] = validRes
					if validRes >= VrBypass {
						break
					}
				}
				if validRes <= VrSilence {
					n.Log.WithField("name", data.Name().String()).Warn("Unverified Data. Drop.")
					ret <- NeedResult{ndn.InterestResultUnverified, nil}
					close(ret)
					return
				}

				// Save (cache) the data in the storage
				deadline := n.engine.Timer().Now()
				if freshness := data.Freshness(); freshness != nil {
					deadline = deadline.Add(*freshness)
				}
				for _, evt := range n.onSaveStorage.val {
					(*evt)(matching, data.Name(), rawData, deadline, context)
				}

				// Return the result
				ret <- NeedResult{ndn.InterestResultData, data.Content()}
				close(ret)
			}()
		})
	if err != nil {
		n.Log.WithField("name", finalName.String()).Warn("Failed to express Interest.")
		ret <- NeedResult{ndn.InterestResultNone, nil}
		close(ret)
		return ret
	}
	return ret
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
	case PropOnSaveStorage:
		return n.onSaveStorage
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
		return PropertySet(&n.canBePrefix, propName, value)
	case PropMustBeFresh:
		return PropertySet(&n.mustBeFresh, propName, value)
	case PropLifetime:
		return PropertySet(&n.lifetime, propName, value)
	case PropIntSigner:
		return PropertySet(&n.intSigner, propName, value)
	case PropSuppressInt:
		return PropertySet(&n.supressInt, propName, value)
	}
	return ndn.ErrNotSupported{Item: string(propName)}
}

func (n *ExpressPoint) Init(parent NTNode, edge enc.ComponentPattern) {
	n.BaseNode.Init(parent, edge)
	n.onInt = NewEvent[*NodeOnIntEvent]()
	n.onValidateInt = NewEvent[*NodeValidateEvent]()
	n.onValidateData = NewEvent[*NodeValidateEvent]()
	n.onSearchStorage = NewEvent[*NodeSearchStorageEvent]()
	n.onSaveStorage = NewEvent[*NodeSaveStorageEvent]()

	n.intSigner = nil
	n.canBePrefix = false // TODO: Set this based on the children
	n.mustBeFresh = true
	n.lifetime = 4 * time.Second
	n.supressInt = false

	n.Self = n
}
