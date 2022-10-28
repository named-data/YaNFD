package schema

import (
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

// CtxKey is the type of keys of the context.
type CtxKey uint64

// Context contains extra info that may be useful to an event, but not worthy to be an argument.
// It can also be used to pass information between chained event callbacks.
type Context map[CtxKey]any

// PropKey is the type of properties of a node.
// A property of a node gives the default setting of some procedure as well as event callbacks.
// We design in this way to support DSL and WASM in future.
type PropKey string

const (
	// The following keys are used to grab extra information of the incoming packet.
	CkDeadline        CtxKey = 1  // Deadline is now + InterestLifetime. [time.Time]
	CkRawPacket       CtxKey = 2  // The raw Interest or Data wire. [enc.Wire]
	CkSigCovered      CtxKey = 3  // The wire covered by the signature. [enc.Wire]
	CkName            CtxKey = 4  // The name of the packet. [enc.Name]
	CkInterest        CtxKey = 5  // The Interest packet. [ndn.Interest]
	CkData            CtxKey = 6  // The Data packet. [ndn.Data]
	CkContent         CtxKey = 7  // The content of the data packet or AppParam of Interest. [enc.Wire]
	CkNackReason      CtxKey = 8  // The NackReason. Only valid when the InterestResult is NACK. [uint64]
	CkLastValidResult CtxKey = 10 // The result returned by the prev validator in the callback chain. [ValidRes]
	CkEngine          CtxKey = 20 // The engine the current node is attached to. [ndn.Engine]

	// The following keys are used to overwrite nodes' properties once
	// Not valid for incoming Interest or Data packets.
	CkCanBePrefix    CtxKey = 101 // [bool]
	CkMustBeFresh    CtxKey = 102 // [bool]
	CkForwardingHint CtxKey = 103 // [[]enc.Name]
	CkNonce          CtxKey = 104 // [time.Duration]
	CkLifetime       CtxKey = 105 // [uint64]
	CkHopLimit       CtxKey = 106 // [uint]
	CkSupressInt     CtxKey = 107 // If true, only local storage is searched. No Interest will be expressed. [bool]
	CkContentType    CtxKey = 201 // [ndn.ContentType]
	CkFreshness      CtxKey = 202 // [time.Duration]
	CkFinalBlockID   CtxKey = 203 // [enc.Component]

	// Whether the data is produced by this program.
	// CkSelfProduced   CtxKey = 204

	// The validity period of a data in the storage produced by this node,
	// i.e. how long the local storage will serve it.
	// Should be larger than FreshnessPeriod. Not affected data fetched remotely.
	CkValidDuration CtxKey = 205 // [time.Duration]
)

// ValidRes is the result of data/interest signature validation, given by one validator.
// When there are multiple validators chained, a packet is valid when there is at least one VrPass and no VrFail.
type ValidRes = int

const (
	VrFail       ValidRes = -2 // An immediate failure. Abort the validation.
	VrTimeout    ValidRes = -1 // Timeout. Abort the validation.
	VrSilence    ValidRes = 0  // The current validator cannot give any result.
	VrPass       ValidRes = 1  // The current validator approves the packet.
	VrBypass     ValidRes = 2  // An immediate success. Bypasses the rest validators that are not executed yet.
	VrCachedData ValidRes = 3  // The data is obtained from a local cache, and the signature is checked before.
)

const (
	// The event called when the node is attached to an engine. [NodeOnAttachEvent]
	PropOnAttach PropKey = "OnAttach"

	// The event called when the node is detached from an engine. [NodeOnDetachEvent]
	PropOnDetach PropKey = "OnDetach"

	// The event called when an ExpressingPoint or LeafNode receives an Interest. [NodeOnIntEvent]
	PropOnInterest PropKey = "OnInterest"

	// The event called when an ExpressingPoint or LeafNode verifies the signature of an Interest. [NodeValidateEvent]
	PropOnValidateInt PropKey = "OnValidateInt"

	// The event called when an ExpressingPoint or LeafNode verifies the signature of a Data. [NodeValidateEvent]
	PropOnValidateData PropKey = "OnValidateData"

	// The event called when an ExpressingPoint or LeafNode searches the storage on Need(). [NodeSearchStorageEvent]
	PropOnSearchStorage PropKey = "OnSearchStorage"

	// The event called when an ExpressingPoint or LeafNode saves a packet into the storage.
	// The packet may be produced locally or fetched from the network.
	// TODO: Add a sign to distinguish them.
	// [NodeSaveStorageEvent]
	PropOnSaveStorage PropKey = "OnSaveStorage"

	// Default CanBePrefix for outgoing Interest. [bool]
	PropCanBePrefix PropKey = "CanBePrefix"
	// Default MustBeFresh for outgoing Interest. [bool]
	PropMustBeFresh PropKey = "MustBeFresh"
	// Default Lifetime for outgoing Interest. [time.Duration]
	PropLifetime PropKey = "Lifetime"
	// Default signer for outgoing Interest. [ndn.Signer]
	PropIntSigner PropKey = "IntSigner"
	// If true, only local storage is searched. No Interest will be expressed.
	// Note: may be overwritten by the Context.
	// [bool]
	PropSuppressInt PropKey = "SupressInt"

	// Default ContentType for produced Data. [ndn.ContentType]
	PropContentType PropKey = "ContentType"
	// Default FreshnessPeriod for produced Data. [time.Duration]
	PropFreshness PropKey = "Freshness"
	// Default signer for produced Data. [ndn.Signer]
	PropDataSigner PropKey = "DataSigner"
	// See CkValidDuration. [time.Duration]
	PropValidDuration PropKey = "ValidDuration"
)

type NodeOnAttachEvent = func(enc.NamePattern, ndn.Engine) error
type NodeOnDetachEvent = func(ndn.Engine)
type NodeOnIntEvent = func(enc.Matching, enc.Wire, ndn.ReplyFunc, Context) bool
type NodeValidateEvent = func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, Context) ValidRes
type NodeSearchStorageEvent = func(enc.Matching, enc.Name, bool, bool, Context) enc.Wire
type NodeSaveStorageEvent = func(enc.Matching, enc.Name, enc.Wire, time.Time, Context)

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
