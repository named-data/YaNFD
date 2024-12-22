package basic_test

import (
	"testing"

	enc "github.com/pulsejet/ndnd/std/encoding"
	basic_engine "github.com/pulsejet/ndnd/std/engine/basic"
	"github.com/pulsejet/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestBasicMatch(t *testing.T) {
	utils.SetTestingT(t)

	var name enc.Name
	var n *basic_engine.NameTrie[int]
	trie := basic_engine.NewNameTrie[int]()

	// Empty match
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	require.True(t, nil == trie.ExactMatch(name))
	require.Equal(t, 0, trie.PrefixMatch(name).Depth())

	// Create /a/b
	name = utils.WithoutErr(enc.NameFromStr("/a/b"))
	n = trie.MatchAlways(name)
	require.Equal(t, 2, n.Depth())
	n.SetValue(10)
	require.Equal(t, 10, n.Value())
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	require.Equal(t, n, trie.PrefixMatch(name))
	require.True(t, nil == trie.ExactMatch(name))

	// First or new will not create /a/b/c
	hasValue := func(x int) bool {
		return x != 0
	}
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	n = trie.FirstSatisfyOrNew(name, hasValue)
	require.Equal(t, 2, n.Depth())

	// MatchAlways will create /a/b/c
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	n = trie.MatchAlways(name)
	require.Equal(t, 3, n.Depth())
	require.Equal(t, 10, n.Parent().Value())

	// Prefix match can reach /a for /a/c
	name = utils.WithoutErr(enc.NameFromStr("/a/c"))
	n = trie.PrefixMatch(name)
	require.Equal(t, 1, n.Depth())

	// First or new will create /a/c
	name = utils.WithoutErr(enc.NameFromStr("/a/c"))
	n = trie.FirstSatisfyOrNew(name, hasValue)
	require.Equal(t, 2, n.Depth())
	require.Equal(t, n, trie.ExactMatch(name))

	// Remove /a/b/c will remove /a/b but not /a/c
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	n = trie.ExactMatch(name)
	n.Delete()
	name = utils.WithoutErr(enc.NameFromStr("/a/b"))
	require.True(t, nil == trie.ExactMatch(name))
	require.Equal(t, 1, trie.PrefixMatch(name).Depth())

	// Remove /a/c will remove everything except the root
	name = utils.WithoutErr(enc.NameFromStr("/a/c"))
	n = trie.ExactMatch(name)
	n.Delete()
	require.False(t, trie.HasChildren())
}

func TestDeleteIf(t *testing.T) {
	utils.SetTestingT(t)

	var name enc.Name
	var n *basic_engine.NameTrie[int]
	trie := basic_engine.NewNameTrie[int]()

	// Create /a/b and /a/b/c
	name = utils.WithoutErr(enc.NameFromStr("/a/b"))
	n = trie.MatchAlways(name)
	n.SetValue(10)
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	n = trie.MatchAlways(name)
	require.Equal(t, 3, n.Depth())
	require.Equal(t, 10, n.Parent().Value())

	noValue := func(x int) bool {
		return x == 0
	}

	// DeleteIf /a/b/c will not remove /a/b
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	n = trie.ExactMatch(name)
	n.DeleteIf(noValue)
	require.True(t, nil == trie.ExactMatch(name))
	name = utils.WithoutErr(enc.NameFromStr("/a/b"))
	require.Equal(t, 10, trie.ExactMatch(name).Value())

	// Create /a/b/c = 10. Now DeleteIf does nothing
	name = utils.WithoutErr(enc.NameFromStr("/a/b/c"))
	n = trie.MatchAlways(name)
	n.SetValue(10)
	n.DeleteIf(noValue)
	require.Equal(t, n, trie.ExactMatch(name))
}
