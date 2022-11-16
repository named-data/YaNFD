package schema

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

// NTNode represents a node (subtree) of NTSchema.
// See comments for NTPolicy below for more details
// TODO: See also BaseNode for TODO points
type NTNode interface {
	// NodeTrait is the type trait of NTNode
	NodeTrait() NTNode

	// Child of given edge
	Child(edge enc.ComponentPattern) NTNode

	// Children include all children of a node
	Children() []NTNode

	// Parent of this node
	Parent() NTNode

	// UpEdge is the edge value from its parent to itself
	UpEdge() enc.ComponentPattern

	// Depth of the node in the tree.
	// It includes the attached prefix, so the root node may have a positive depth
	Depth() uint

	// Match an NDN name to a (variable) matching
	// TODO: do people like the current matching or a simpler version that makes everything into []byte?
	Match(name enc.Name) (NTNode, enc.Matching)

	// Apply a (variable) matching and obtain the corresponding NDN name
	Apply(matching enc.Matching) enc.Name

	// Get a property or callback event
	// TODO: reflection may be better than manual implementation?
	Get(propName PropKey) any

	// Set a property
	// Please use Get to set callback event.
	// To avoid annoying type cast, consider use AddEventListener to help.
	Set(propName PropKey, value any) error

	// ConstructName is the aux function used by Apply
	ConstructName(matching enc.Matching, ret enc.Name) error

	// OnInterest is the callback function when there is an incoming Interest.
	OnInterest(interest ndn.Interest, rawInterest enc.Wire, sigCovered enc.Wire,
		reply ndn.ReplyFunc, deadline time.Time, matching enc.Matching)

	// OnAttach is called when the node is attached to an engine
	OnAttach(path enc.NamePattern, engine ndn.Engine) error

	// OnDetach is called when the node is detached from an engine
	OnDetach()

	// At gets a node/subtree at a given pattern path. The path does not include the attached prefix.
	At(path enc.NamePattern) NTNode

	// PutNode puts a node/subtree at a given pattern path. The path does not include the attached prefix.
	// The path must not be put with an existing node.
	// The node will be initialized after PutNode.
	// Note: the correct sig of this func should be Put[T NTNode](path), but golang provents this.
	PutNode(path enc.NamePattern, node NTNode) error

	// Init initializes a node with a specified parent and edge
	// This is the first part where one can modify the properties of the node.
	// It acts as the constructor, so the properties set here are the default values, and will be
	// overwriten by the policies' apply().
	Init(parent NTNode, edge enc.ComponentPattern)

	// AttachedPrefix returns the attached prefix if this node is the root of a schema tree.
	// Return nil otherwise.
	AttachedPrefix() enc.Name

	// SetAttachedPrefix set the attached prefix if the node is the root of a schema tree.
	SetAttachedPrefix(enc.Name) error
}

// NTPolicy represents a policy, which is a modifier that sets properties and registers events
// during the initialization of a schema tree.
// The execution order is: construct the tree -> apply policies & env setup -> attach to engine (see Tree)
// Though policies are also considered as static knowledge (at compile time),
// they may be configured differently on different nodes.
// For example, an storage may access different folders for different instances.
// TODO: Design for this configuration under different environments.
// For example, the path to a storage, the name of the self-key, etc are knowledge specific to one instance,
// and thus dynamically configured.
// Currently I don't have an idea on the best way to separate the "statically shared part" (NTTree and the nodes)
// and the "dynamically configure part". (#ENV)
type NTPolicy interface {
	// PolicyTrait is the type trait of NTPolicy
	PolicyTrait() NTPolicy

	// Apply the policy at a node (subtree). May modify the children of the given node.
	// The execution order is node.Init() -> policy.Apply() -> onAttach()
	// So the properties set here will overwrite Init()'s default values,
	// but be overwriten by onAttach().
	Apply(node NTNode) error
}

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
	// The following keys are used to grab extra information of the incoming packet. [TYPE]
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

// NodeOnAttachEvent is the event triggered when attaching to an engine.
// This is called after the tree is built and policies are attached,
// so the properties modified here will overwrite the properties set before.
// Args: the path to this node, the engine to attach
// Return: a non-nil error to abort attaching. Nil when it can continue.
type NodeOnAttachEvent = func(enc.NamePattern, ndn.Engine) error

// NodeOnDetachEvent is the event triggered when detaching from the engine.
type NodeOnDetachEvent = func(ndn.Engine)

// NodeOnIntEvent is the event triggered when an Interest is received.
// Only used by ExpressingPoint and LeafNode.
// Args: name matching, app params, reply, context
// Return: whether to abort processing (ignore events after the current one)
// Context: CkInterest, CkDeadline, CkRawPacket, CkSigCovered, CkName, CkEngine, CkContent(=AppParam)
type NodeOnIntEvent = func(enc.Matching, enc.Wire, ndn.ReplyFunc, Context) bool

// NodeValidateEvent is the event triggered when the node needs to validate an Interest or a Data
// Used by ExpressingPoint and LeafNode.
// Args: name matching, full name, signature, covered part, context
// Return: the result of current validation process
// Context: CkInterest/CkData, CkName, CkDeadline, CkRawPacket, CkSigCovered, CkEngine, CkContent(=Content/AppParam)
// CkLastValidResult (the VrResult given by the last callback in the chain)
type NodeValidateEvent = func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, Context) ValidRes

// NodeSearchStorageEvent is the event triggered when the node searches the storage for data.
// Used by ExpressingPoint and LeafNode.
// Args: name matching, full name, can be prefix, must be fresh, context
// Return: the raw wire of Data packet. Nil if not existing
// Context: CkInterest, CkDeadline, CkRawPacket, CkSigCovered, CkName, CkEngine, CkContent(=AppParam)
type NodeSearchStorageEvent = func(enc.Matching, enc.Name, bool, bool, Context) enc.Wire

// NodeSaveStorageEvent is the event triggered when the node stores the Data into the storage.
// Args: name matching, full name, raw data packet, validity time (See CkValidDuration), context
// Context: CkData, CkRawPacket, CkSigCovered, CkName, CkEngine, CkContent
type NodeSaveStorageEvent = func(enc.Matching, enc.Name, enc.Wire, time.Time, Context)
