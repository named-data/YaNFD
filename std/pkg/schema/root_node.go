// Abandoned code. For remark only. A specific root node may cause trouble.
package schema

// import (
// 	"errors"
// 	"time"

// 	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
// 	"github.com/zjkmxy/go-ndn/pkg/ndn"
// )

// type RootNode struct {
// 	BaseNode

// 	Prefix enc.Name
// }

// func (n *RootNode) Attach(prefix enc.Name, engine ndn.Engine) error {
// 	n.Prefix = prefix
// 	path := make(enc.NamePattern, len(prefix))
// 	for i, c := range prefix {
// 		path[i] = c
// 	}
// 	err := n.OnAttach(path, engine)
// 	if err != nil {
// 		return err
// 	}
// 	n.Log.Info("Attached to engine.")
// 	return engine.AttachHandler(prefix, n.intHandler)
// }

// func (n *RootNode) Detach() {
// 	n.Log.Info("Detached from engine")
// 	n.engine.DetachHandler(n.Prefix)
// 	n.OnDetach()
// }

// // Parent of this node
// func (n *RootNode) Parent() NTNode {
// 	return nil
// }

// // UpEdge is the edge value from its parent to itself
// func (n *RootNode) UpEdge() enc.ComponentPattern {
// 	return nil
// }

// // Match an NDN name to a (variable) matching
// func (n *RootNode) Match(name enc.Name) (NTNode, enc.Matching) {
// 	if !n.Prefix.IsPrefix(name) {
// 		return nil, nil
// 	}
// 	return n.BaseNode.Match(name[n.Dep:])
// }

// // ConstructName is the aux function used by Apply
// func (n *RootNode) ConstructName(matching enc.Matching, ret enc.Name) error {
// 	if len(ret) < int(n.Dep) {
// 		return errors.New("insufficient name length") // This error won't be returned to the user
// 	}
// 	copy(ret[:n.Dep], n.Prefix)
// 	return nil
// }

// func (n *RootNode) intHandler(
// 	interest ndn.Interest, rawInterest enc.Wire, sigCovered enc.Wire,
// 	reply ndn.ReplyFunc, deadline time.Time,
// ) {
// 	matchName := interest.Name()
// 	if matchName[len(matchName)-1].Typ == enc.TypeParametersSha256DigestComponent {
// 		matchName = matchName[:len(matchName)-1]
// 	}
// 	node, matching := n.Match(matchName)
// 	if node == nil {
// 		n.Log.WithField("name", interest.Name().String()).Warn("Unexpected Interest. Drop.")
// 		return
// 	}
// 	node.OnInterest(interest, rawInterest, sigCovered, reply, deadline, matching)
// }
