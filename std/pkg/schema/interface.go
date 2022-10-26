package schema

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

// NTNode represents a subtree of NTSchema.
type NTNode interface {
	// NodeTrait is the type trait of NTNode
	NodeTrait() NTNode

	// Child of given edge
	Child(edge enc.ComponentPattern) NTNode

	// Parent of this node
	Parent() NTNode

	// UpEdge is the edge value from its parent to itself
	UpEdge() enc.ComponentPattern

	// Depth of the node in the tree.
	// It includes the attached prefix, so the root node may have a positive depth
	Depth() uint

	// Match an NDN name to a (variable) matching
	Match(name enc.Name) (NTNode, enc.Matching)

	// Apply a (variable) matching and obtain the corresponding NDN name
	Apply(matching enc.Matching) enc.Name

	// Get a property or callback event
	Get(propName PropKey) any

	// Set a property or callback event
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

	// PutNode(path enc.NamePattern, nodeCons NodeConstructor) error

	// Init initializes a node with a specified parent and edge
	Init(parent NTNode, edge enc.ComponentPattern)

	// AttachedPrefix returns the attached prefix if this node is the root of a schema tree.
	// Return nil otherwise.
	AttachedPrefix() enc.Name

	// SetAttachedPrefix set the attached prefix if the node is the root of a schema tree.
	SetAttachedPrefix(enc.Name) error
}

type NodeConstructor = func(parent NTNode, edge enc.ComponentPattern) NTNode
