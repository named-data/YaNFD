package schema

import (
	"errors"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

type BaseNode struct {
	Self NTNode
	Chd  []NTNode
	Log  *log.Entry

	dep  uint
	par  NTNode
	edge enc.ComponentPattern

	attachedPrefix enc.Name

	engine ndn.Engine

	onAttachEvt *Event[*NodeOnAttachEvent]
	onDetachEvt *Event[*NodeOnDetachEvent]
}

// Note: Inheritance from BaseNode is really a bad model of this thing
// but I cannot come up with a better one in limited time.
// If possible, a mixin programming model may be better.

// NodeTrait is the type trait of NTNode
func (n *BaseNode) NodeTrait() NTNode {
	return n
}

// Child of given edge
func (n *BaseNode) Child(edge enc.ComponentPattern) NTNode {
	for _, c := range n.Chd {
		if c.UpEdge().Equal(edge) {
			return c
		}
	}
	return nil
}

// Parent of this node
func (n *BaseNode) Parent() NTNode {
	return n.par
}

// UpEdge is the edge value from its parent to itself
func (n *BaseNode) UpEdge() enc.ComponentPattern {
	return n.edge
}

// Depth of the node in the tree.
// It includes the attached prefix, so the root node may have a positive depth
func (n *BaseNode) Depth() uint {
	return n.dep
}

// Match an NDN name to a (variable) matching
func (n *BaseNode) Match(name enc.Name) (NTNode, enc.Matching) {
	if len(n.attachedPrefix) > 0 {
		// Only happens when n is the root node
		if !n.attachedPrefix.IsPrefix(name) {
			return nil, nil
		}
		name = name[n.dep:]
	}
	if len(name) <= 0 {
		return n.Self, make(enc.Matching)
	}
	for _, c := range n.Chd {
		if c.UpEdge().IsMatch(name[0]) {
			dst, mat := c.Match(name[1:])
			if dst == nil {
				return nil, nil
			} else {
				c.UpEdge().Match(name[0], mat)
				return dst, mat
			}
		}
	}
	return nil, nil
}

// Apply a (variable) matching and obtain the corresponding NDN name
func (n *BaseNode) Apply(matching enc.Matching) enc.Name {
	ret := make(enc.Name, n.dep)
	if n.ConstructName(matching, ret) == nil {
		return ret
	} else {
		return nil
	}
}

// ConstructName is the aux function used by Apply
func (n *BaseNode) ConstructName(matching enc.Matching, ret enc.Name) error {
	if n.par == nil {
		if len(ret) < int(n.dep) {
			return errors.New("insufficient name length") // This error won't be returned to the user
		}
		copy(ret[:n.dep], n.attachedPrefix)
		return nil
	} else {
		c, err := n.edge.FromMatching(matching)
		if err != nil {
			return err
		}
		ret[n.dep-1] = *c
		return n.par.ConstructName(matching, ret)
	}
}

// OnInterest is the function called when an Interest comes.
// A base node shouldn't receive any Interest, so drops it.
func (n *BaseNode) OnInterest(
	interest ndn.Interest, rawInterest enc.Wire, sigCovered enc.Wire,
	reply ndn.ReplyFunc, deadline time.Time, matching enc.Matching,
) {
	n.Log.WithField("name", interest.Name().String()).Warn("Unexpected Interest. Drop.")
}

// OnAttach is called when the node is attached to an engine
// BaseNode will call the event set by policy
func (n *BaseNode) OnAttach(path enc.NamePattern, engine ndn.Engine) error {
	n.engine = engine
	n.dep = uint(len(path))
	n.Log = log.WithField("module", "schema").WithField("path", path.String())
	for _, evt := range n.onAttachEvt.val {
		err := (*evt)(path, engine)
		if err != nil {
			n.Log.Errorf("Attaching failed with error: %+v", err)
			return err
		}
	}
	for _, c := range n.Chd {
		nxtPath := append(path, c.UpEdge())
		err := c.OnAttach(nxtPath, engine)
		if err != nil {
			return err
		}
	}
	return nil
}

// OnDetach is called when the node is detached from an engine
// BaseNode will call the event set by policy
func (n *BaseNode) OnDetach() {
	for _, c := range n.Chd {
		c.OnDetach()
	}
	for _, evt := range n.onDetachEvt.val {
		(*evt)(n.engine)
	}
	n.engine = nil
}

// Get a property or callback event
func (n *BaseNode) Get(propName PropKey) any {
	switch propName {
	case PropOnAttach:
		return n.onAttachEvt
	case PropOnDetach:
		return n.onDetachEvt
	}
	return nil
}

// Set a property. Use Get() to update callback events.
func (n *BaseNode) Set(propName PropKey, value any) error {
	return ndn.ErrNotSupported{Item: string(propName)}
}

// At gets a node/subtree at a given pattern path. The path does not include the attached prefix.
func (n *BaseNode) At(path enc.NamePattern) NTNode {
	if len(path) <= 0 {
		return n.Self
	}
	for _, c := range n.Chd {
		if c.UpEdge().Equal(path[0]) {
			return c.At(path[1:])
		}
	}
	return nil
}

// PutNode sets a node/subtree at a given pattern path. The path does not include the attached prefix.
func (n *BaseNode) PutNode(path enc.NamePattern, node NTNode) error {
	if len(path) <= 0 {
		return errors.New("schema node already exists")
	}
	for _, c := range n.Chd {
		if c.UpEdge().Equal(path[0]) {
			return c.PutNode(path[1:], node)
		}
	}
	if len(path) > 1 {
		// In this case, node is not our direct child
		nxtChd := &BaseNode{}
		nxtChd.Init(n.Self, path[0])
		n.Chd = append(n.Chd, nxtChd)
		return nxtChd.PutNode(path[1:], node)
	} else {
		n.Chd = append(n.Chd, node)
		node.Init(n.Self, path[0])
	}
	return nil
}

func (n *BaseNode) Init(parent NTNode, edge enc.ComponentPattern) {
	*n = BaseNode{
		dep:            0,
		par:            parent,
		edge:           edge,
		attachedPrefix: nil,
		Chd:            make([]NTNode, 0),
		Log:            nil,
		engine:         nil,
		onAttachEvt:    NewEvent[*NodeOnAttachEvent](),
		onDetachEvt:    NewEvent[*NodeOnDetachEvent](),
		Self:           n,
	}
}

func (n *BaseNode) AttachedPrefix() enc.Name {
	return n.attachedPrefix
}

func (n *BaseNode) SetAttachedPrefix(prefix enc.Name) error {
	if n.par == nil {
		n.attachedPrefix = prefix
		return nil
	} else {
		return errors.New("only root nodes are attachable")
	}
}

func (n *BaseNode) Children() []NTNode {
	return n.Chd
}
