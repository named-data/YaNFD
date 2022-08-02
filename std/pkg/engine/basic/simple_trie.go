package basic

import enc "github.com/zjkmxy/go-ndn/pkg/encoding"

// NameTrie is a simple implementation of a Name trie (node/subtree) used for PIT and FIB.
// It is slow due to the usage of String(). Subject to change when it explicitly affects performance.
type NameTrie[V any] struct {
	val V
	key string
	par *NameTrie[V]
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
	if len(name) == 0 {
		return n
	}
	c := name[0].String()
	if ch, ok := n.chd[c]; ok {
		return ch.ExactMatch(name[1:])
	} else {
		return nil
	}
}

// LongMatch returns the longest prefix match of the name. Always succeeds.
func (n *NameTrie[V]) LongMatch(name enc.Name) *NameTrie[V] {
	if len(name) == 0 {
		return n
	}
	c := name[0].String()
	if ch, ok := n.chd[c]; ok {
		return ch.LongMatch(name[1:])
	} else {
		return n
	}
}

// newTrieNode creates a new NameTrie node.
func newTrieNode[V any](key string, parent *NameTrie[V]) *NameTrie[V] {
	return &NameTrie[V]{
		par: parent,
		chd: map[string]*NameTrie[V]{},
		key: key,
	}
}

// MatchAlways finds or creates the node that matches the name exactly.
func (n *NameTrie[V]) MatchAlways(name enc.Name) *NameTrie[V] {
	if len(name) == 0 {
		return n
	}
	c := name[0].String()
	ch, ok := n.chd[c]
	if !ok {
		ch = newTrieNode(c, n)
		n.chd[c] = ch
	}
	return ch.MatchAlways(name[1:])
}

// FirstSatisfyOrNew finds or creates the first node along the path that satisfies the predicate.
func (n *NameTrie[V]) FirstSatisfyOrNew(name enc.Name, pred func(V) bool) *NameTrie[V] {
	if len(name) == 0 || pred(n.val) {
		return n
	}
	c := name[0].String()
	ch, ok := n.chd[c]
	if !ok {
		ch = newTrieNode(c, n)
		n.chd[c] = ch
	}
	return ch.FirstSatisfyOrNew(name[1:], pred)
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

// NewNameTrie creates a new NameTrie and returns the root node.
func NewNameTrie[V any]() *NameTrie[V] {
	return newTrieNode[V]("", nil)
}
