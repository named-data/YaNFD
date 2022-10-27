package schema

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type LeafNode struct {
	ExpressPoint

	dataSigner  ndn.Signer
	contentType ndn.ContentType
	freshness   time.Duration
	validDur    time.Duration
}

// TODO: make sure code handles when context or matching is nil

func (n *LeafNode) Provide(
	matching enc.Matching, name enc.Name, content enc.Wire, context Context,
) enc.Wire {
	// Construct the name if not yet
	if name == nil {
		name = n.Apply(matching)
		if name == nil {
			n.Log.Error("Unable to construct Data Name in Provide().")
			return nil
		}
	}

	// Construst the Data
	engine := n.engine
	spec := engine.Spec()
	signer := n.dataSigner
	dataCfg := ndn.DataConfig{
		ContentType:  utils.IdPtr(n.contentType),
		Freshness:    utils.IdPtr(n.freshness),
		FinalBlockID: nil,
	}
	validDur := n.validDur
	if ctxVal, ok := context[CkContentType]; ok {
		if v, ok := ctxVal.(ndn.ContentType); ok {
			dataCfg.ContentType = &v
		}
	}
	if ctxVal, ok := context[CkFreshness]; ok {
		if v, ok := ctxVal.(time.Duration); ok {
			dataCfg.Freshness = &v
		}
	}
	if ctxVal, ok := context[CkFinalBlockID]; ok {
		if v, ok := ctxVal.(*enc.Component); ok {
			dataCfg.FinalBlockID = v
		}
	}
	if v, ok := context[CkValidDuration].(time.Duration); ok {
		validDur = v
	}
	wire, _, err := spec.MakeData(name, &dataCfg, content, signer)
	if err != nil {
		n.Log.Errorf("Unable to encode Data in Provide(): %+v", err)
		return nil
	}

	// Store data in the storage
	context[CkEngine] = n.engine
	deadline := n.engine.Timer().Now().Add(validDur)
	for _, evt := range n.onSaveStorage.val {
		(*evt)(matching, name, wire, deadline, context)
	}

	// Return encoded data
	return wire
}

// Get a property or callback event
func (n *LeafNode) Get(propName PropKey) any {
	if ret := n.ExpressPoint.Get(propName); ret != nil {
		return ret
	}
	switch propName {
	case PropContentType:
		return n.contentType
	case PropFreshness:
		return n.freshness
	case PropDataSigner:
		return n.dataSigner
	case PropValidDuration:
		return n.validDur
	}
	return nil
}

// Set a property. Use Get() to update callback events.
func (n *LeafNode) Set(propName PropKey, value any) error {
	if ret := n.ExpressPoint.Set(propName, value); ret == nil {
		return ret
	}
	switch propName {
	case PropContentType:
		return PropertySet(&n.contentType, propName, value)
	case PropFreshness:
		return PropertySet(&n.freshness, propName, value)
	case PropDataSigner:
		return PropertySet(&n.dataSigner, propName, value)
	case PropValidDuration:
		return PropertySet(&n.validDur, propName, value)
	}
	return ndn.ErrNotSupported{Item: string(propName)}
}

func (n *LeafNode) Init(parent NTNode, edge enc.ComponentPattern) {
	n.ExpressPoint.Init(parent, edge)

	n.dataSigner = nil
	n.contentType = ndn.ContentTypeBlob
	n.freshness = 0

	n.Self = n
}
