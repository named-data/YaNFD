package basic

import enc "github.com/zjkmxy/go-ndn/pkg/encoding"

// NameTrie is a simple implementation of a Name trie (node/subtree) used for PIT and FIB.
// It is slow due to the usage of String(). Subject to change when it explicitly affects performance.
type NameTrie[V any] struct {
	val V
	key string
	par *NameTrie[V]
	dep int
	chd map[string]*NameTrie[V]
}

// Value returns the value stored in the node.
func (n *NameTrie[V]) Value() V {
	return n.val
}

// SetValue puts some value in the node.
func (n *NameTrie[V]) SetValue(value V) {
	n.val = value
}

// ExactMatch returns the node that matches the name exactly. If no node matches, it returns nil.
func (n *NameTrie[V]) ExactMatch(name enc.Name) *NameTrie[V] {
	if len(name) <= n.dep {
		return n
	}
	c := name[n.dep].String()
	if ch, ok := n.chd[c]; ok {
		return ch.ExactMatch(name)
	} else {
		return nil
	}
}

// PrefixMatch returns the longest prefix match of the name.
// Always succeeds, but the returned node may be empty.
func (n *NameTrie[V]) PrefixMatch(name enc.Name) *NameTrie[V] {
	if len(name) <= n.dep {
		return n
	}
	c := name[n.dep].String()
	if ch, ok := n.chd[c]; ok {
		return ch.PrefixMatch(name)
	} else {
		return n
	}
}

// newTrieNode creates a new NameTrie node.
func newTrieNode[V any](key string, parent *NameTrie[V]) *NameTrie[V] {
	depth := 0
	if parent != nil {
		depth = parent.dep + 1
	}
	return &NameTrie[V]{
		par: parent,
		chd: map[string]*NameTrie[V]{},
		key: key,
		dep: depth,
	}
}

// MatchAlways finds or creates the node that matches the name exactly.
func (n *NameTrie[V]) MatchAlways(name enc.Name) *NameTrie[V] {
	if len(name) <= n.dep {
		return n
	}
	c := name[n.dep].String()
	ch, ok := n.chd[c]
	if !ok {
		ch = newTrieNode(c, n)
		n.chd[c] = ch
	}
	return ch.MatchAlways(name)
}

// FirstSatisfyOrNew finds or creates the first node along the path that satisfies the predicate.
func (n *NameTrie[V]) FirstSatisfyOrNew(name enc.Name, pred func(V) bool) *NameTrie[V] {
	if len(name) <= n.dep || pred(n.val) {
		return n
	}
	c := name[n.dep].String()
	ch, ok := n.chd[c]
	if !ok {
		ch = newTrieNode(c, n)
		n.chd[c] = ch
	}
	return ch.FirstSatisfyOrNew(name, pred)
}

// HasChildren returns whether the node has children.
func (n *NameTrie[V]) HasChildren() bool {
	return len(n.chd) > 0
}

// Delete deletes the node itself. Altomatically removes the parent node if it is empty.
func (n *NameTrie[V]) Delete() {
	if n.par != nil {
		n.chd = nil
		delete(n.par.chd, n.key)
		if len(n.par.chd) == 0 {
			n.par.Delete()
		}
	} else {
		// Root node cannot be deleted.
		n.chd = map[string]*NameTrie[V]{}
	}
}

// Depth returns the depth of a node in the tree.
func (n *NameTrie[V]) Depth() int {
	return n.dep
}

// Parent returns its parent node.
func (n *NameTrie[V]) Parent() *NameTrie[V] {
	return n.par
}

// NewNameTrie creates a new NameTrie and returns the root node.
func NewNameTrie[V any]() *NameTrie[V] {
	return newTrieNode[V]("", nil)
}
