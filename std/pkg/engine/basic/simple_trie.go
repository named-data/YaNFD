package basic

// NameTrie is a simple implementation of a Name trie (node/subtree) used for PIT and FIB.
// It is slow due to the usage of String(). Subject to change when it explicitly affects performance.
type NameTrie[V any] struct {
	Val V
	Chd map[string]*NameTrie[V]
}

// For FIB: easy prefix match.
// For PIT: How to handle can be prefix?
// Maybe just trie but let FIB and PIT function to be handled by the engine.
