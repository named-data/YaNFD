package schema

import (
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

type CtxKey uint64
type Context map[CtxKey]any

type PropKey string // Reserved for now

const (
	CkDeadline        CtxKey = 1
	CkRawPacket       CtxKey = 2
	CkSigCovered      CtxKey = 3
	CkName            CtxKey = 4
	CkInterest        CtxKey = 5
	CkData            CtxKey = 6
	CkContent         CtxKey = 7
	CkNackReason      CtxKey = 8
	CkLastValidResult CtxKey = 10
	CkEngine          CtxKey = 20

	// The following keys are used to overwrite nodes' properties once
	// Not valid for incoming Interest or Data packets.
	CkCanBePrefix    CtxKey = 101
	CkMustBeFresh    CtxKey = 102
	CkForwardingHint CtxKey = 103
	CkNonce          CtxKey = 104
	CkLifetime       CtxKey = 105
	CkHopLimit       CtxKey = 106
	CkSupressInt     CtxKey = 107
	CkContentType    CtxKey = 201
	CkFreshness      CtxKey = 202
	CkFinalBlockID   CtxKey = 203
	CkSelfProduced   CtxKey = 204
	CkValidDuration  CtxKey = 205
)

type ValidRes = int

const (
	VrFail       ValidRes = -2
	VrTimeout    ValidRes = -1
	VrSilence    ValidRes = 0
	VrPass       ValidRes = 1
	VrBypass     ValidRes = 2
	VrCachedData ValidRes = 3
)

const (
	PropOnAttach        PropKey = "OnAttach"
	PropOnDetach        PropKey = "OnDetach"
	PropOnInterest      PropKey = "OnInterest"
	PropOnValidateInt   PropKey = "OnValidateInt"
	PropOnValidateData  PropKey = "OnValidateData"
	PropOnSearchStorage PropKey = "OnSearchStorage"
	PropOnSaveStorage   PropKey = "OnSaveStorage"

	PropCanBePrefix PropKey = "CanBePrefix"
	PropMustBeFresh PropKey = "MustBeFresh"
	PropLifetime    PropKey = "Lifetime"
	PropIntSigner   PropKey = "IntSigner"
	PropSuppressInt PropKey = "SupressInt"

	PropContentType   PropKey = "ContentType"
	PropFreshness     PropKey = "Freshness"
	PropDataSigner    PropKey = "DataSigner"
	PropValidDuration PropKey = "ValidDuration"
)

type NodeOnAttachEvent = func(enc.NamePattern, ndn.Engine) error
type NodeOnDetachEvent = func(ndn.Engine)
type NodeOnIntEvent = func(enc.Matching, enc.Wire, ndn.ReplyFunc, Context) bool
type NodeValidateEvent = func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, Context) ValidRes
type NodeSearchStorageEvent = func(enc.Matching, enc.Name, bool, bool, Context) enc.Wire
type NodeSaveStorageEvent = func(enc.Matching, enc.Name, enc.Wire, time.Time, Context)

// type NodePreSendDataEvent = func(enc.Matching, enc.Wire, Context)
// type NodePreSendIntEvent = func(enc.Matching, enc.Wire, Context)
// type NodePreRecvDataEvent = func(enc.Matching, enc.Wire, Context)
// type NodePreRecvIntEvent = func(enc.Matching, enc.Wire, Context)

func PropertySet[T any](ptr *T, propName PropKey, value any) error {
	if v, ok := value.(T); ok {
		*ptr = v
		return nil
	} else {
		return ndn.ErrInvalidValue{Item: string(propName), Value: value}
	}
}

func AddEventListener[T any](node NTNode, propName PropKey, callback T) error {
	evt, ok := node.Get(propName).(*Event[*T])
	if !ok || evt == nil {
		return fmt.Errorf("invalid event: %s", propName)
	}
	evt.Add(&callback)
	return nil
}

// func New[T NTNode](parent NTNode, edge enc.ComponentPattern) NTNode {
// 	ret := NTNode(new(T))
// 	ret.Init(parent, edge)
// 	return ret
// }
