/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2022 Danning Yu.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"sort"
	"testing"

	"github.com/named-data/YaNFD/ndn"

	"github.com/stretchr/testify/assert"
)

func TestFindNextHops_HT(t *testing.T) {
	fibTableAlgorithm = "hashtable"
	newFibStrategyTableHashTable(1)

	assert.NotNil(t, FibStrategyTable)

	// Root entry has no hops
	name1, _ := ndn.NameFromString("/")
	nexthops1 := FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nexthops1))

	// Next hops need to be explicitly added
	name2, _ := ndn.NameFromString("/test")
	nexthops2 := FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 0, len(nexthops2))
	FibStrategyTable.InsertNextHop(name2, 25, 1)
	FibStrategyTable.InsertNextHop(name2, 101, 10)
	nexthops2a := FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 2, len(nexthops2a))
	assert.Equal(t, uint64(25), nexthops2a[0].Nexthop)
	assert.Equal(t, uint64(1), nexthops2a[0].Cost)
	assert.Equal(t, uint64(101), nexthops2a[1].Nexthop)
	assert.Equal(t, uint64(10), nexthops2a[1].Cost)

	// Check longest prefix match, should match with /test
	// and then return its next hops
	name3, _ := ndn.NameFromString("/test/name/202=abc123")
	nexthops3 := FibStrategyTable.FindNextHops(name3)
	assert.Equal(t, 2, len(nexthops3))
	assert.Equal(t, uint64(25), nexthops3[0].Nexthop)
	assert.Equal(t, uint64(1), nexthops3[0].Cost)
	assert.Equal(t, uint64(101), nexthops3[1].Nexthop)
	assert.Equal(t, uint64(10), nexthops3[1].Cost)
	nexthops1a := FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nexthops1a))

	// Next hops should be updated when they're removed
	FibStrategyTable.RemoveNextHop(name2, 25)
	nexthops2b := FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 1, len(nexthops2b))
	assert.Equal(t, uint64(101), nexthops2b[0].Nexthop)
	assert.Equal(t, uint64(10), nexthops2b[0].Cost)

	// Test pruning
	name4, _ := ndn.NameFromString("/test4")
	name5, _ := ndn.NameFromString("/test5")
	FibStrategyTable.InsertNextHop(name4, 25, 1)
	FibStrategyTable.InsertNextHop(name5, 25, 1)

	FibStrategyTable.RemoveNextHop(name4, 25)
	nexthops2c := FibStrategyTable.FindNextHops(name4)
	assert.Equal(t, 0, len(nexthops2c))
}

func TestFind_Set_Unset_Strategy_HT(t *testing.T) {
	newFibStrategyTableHashTable(1)

	assert.NotNil(t, FibStrategyTable)

	bestRoute, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := ndn.NameFromString("/localhost/nfd/strategy/multicast/v=1")

	name1, _ := ndn.NameFromString("/")
	assert.True(t, bestRoute.Equals(FibStrategyTable.FindStrategy(name1)))

	name2, _ := ndn.NameFromString("/test")
	assert.True(t, bestRoute.Equals(FibStrategyTable.FindStrategy(name2)))
	FibStrategyTable.SetStrategy(name2, multicast)
	assert.True(t, bestRoute.Equals(FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name2)))

	name3, _ := ndn.NameFromString("/test/name/202=abc123")
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name3)))
	FibStrategyTable.SetStrategy(name3, bestRoute)
	assert.True(t, bestRoute.Equals(FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name2)))
	assert.True(t, bestRoute.Equals(FibStrategyTable.FindStrategy(name3)))

	// Test pruning
	FibStrategyTable.UnsetStrategy(name3)
	assert.True(t, bestRoute.Equals(FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name2)))
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name3)))

	FibStrategyTable.SetStrategy(name1, multicast)
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name2)))
	assert.True(t, multicast.Equals(FibStrategyTable.FindStrategy(name3)))
}

func TestInsertNextHop_HT(t *testing.T) {
	newFibStrategyTableHashTable(1)
	assert.NotNil(t, FibStrategyTable)

	name, _ := ndn.NameFromString("/test/name")

	// Insert new hop
	FibStrategyTable.InsertNextHop(name, 100, 10)
	nextHops := FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, uint64(100), nextHops[0].Nexthop)
	assert.Equal(t, uint64(10), nextHops[0].Cost)

	// Update cost of current hop
	FibStrategyTable.InsertNextHop(name, 100, 20)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, uint64(100), nextHops[0].Nexthop)
	assert.NotEqual(t, uint64(10), nextHops[0].Cost)
	assert.Equal(t, uint64(20), nextHops[0].Cost)
}

func TestClearNextHops_HT(t *testing.T) {
	newFibStrategyTableHashTable(1)
	assert.NotNil(t, FibStrategyTable)

	name, _ := ndn.NameFromString("/test/name")

	// Insert new hop
	FibStrategyTable.InsertNextHop(name, 100, 10)
	FibStrategyTable.InsertNextHop(name, 100, 20)
	FibStrategyTable.InsertNextHop(name, 200, 10)
	FibStrategyTable.InsertNextHop(name, 300, 10)

	nextHops := FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 3, len(nextHops))

	FibStrategyTable.ClearNextHops(name)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 0, len(nextHops))

	// Should have no effect on a name with no hops
	// Or an nonexistent name
	FibStrategyTable.ClearNextHops(name)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 0, len(nextHops))

	nameDoesNotExist, _ := ndn.NameFromString("/asdf")
	FibStrategyTable.ClearNextHops(nameDoesNotExist)
	nextHops = FibStrategyTable.FindNextHops(nameDoesNotExist)
	assert.Equal(t, 0, len(nextHops))

	// Should only clear hops for exact match
	FibStrategyTable.InsertNextHop(name, 100, 10)
	nameLonger, _ := ndn.NameFromString("/test/name/longer")
	FibStrategyTable.InsertNextHop(nameLonger, 200, 10)

	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 1, len(nextHops))
	FibStrategyTable.ClearNextHops(name)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 0, len(nextHops))

	nextHops = FibStrategyTable.FindNextHops(nameLonger)
	assert.Equal(t, 1, len(nextHops))
}

func TestRemoveNextHop_HT(t *testing.T) {
	newFibStrategyTableHashTable(1)
	assert.NotNil(t, FibStrategyTable)

	name, _ := ndn.NameFromString("/test")

	// Insert new hop
	hopId1 := uint64(100)
	hopId2 := uint64(200)
	hopId3 := uint64(300)
	FibStrategyTable.InsertNextHop(name, hopId1, 10)
	FibStrategyTable.InsertNextHop(name, hopId2, 10)
	FibStrategyTable.InsertNextHop(name, hopId3, 10)
	FibStrategyTable.InsertNextHop(name, hopId1, 20) // updates it in place

	nextHops := FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 3, len(nextHops))

	FibStrategyTable.RemoveNextHop(name, hopId1)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 2, len(nextHops))

	FibStrategyTable.RemoveNextHop(name, hopId2)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 1, len(nextHops))

	FibStrategyTable.RemoveNextHop(name, hopId3)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 0, len(nextHops))

	FibStrategyTable.InsertNextHop(name, hopId1, 10)
	nameLonger, _ := ndn.NameFromString("/test/name/longer")
	FibStrategyTable.InsertNextHop(nameLonger, hopId2, 10)

	FibStrategyTable.RemoveNextHop(name, hopId1)
	nextHops = FibStrategyTable.FindNextHops(name)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops(nameLonger)
	assert.Equal(t, 1, len(nextHops))
}

func TestGetAllFIBEntries_HT(t *testing.T) {
	newFibStrategyTableHashTable(1)
	assert.NotNil(t, FibStrategyTable)

	bestRoute, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := ndn.NameFromString("/localhost/nfd/strategy/multicast/v=1")

	hopId2 := uint64(200)
	hopId3 := uint64(300)

	// Only strategy, no next hops, so it shouldn't be returned
	name, _ := ndn.NameFromString("/test")
	FibStrategyTable.SetStrategy(name, multicast)

	name2, _ := ndn.NameFromString("/test/name/202=abc123")
	FibStrategyTable.SetStrategy(name2, bestRoute)
	FibStrategyTable.InsertNextHop(name2, hopId2, 20)
	FibStrategyTable.InsertNextHop(name2, hopId3, 30)

	// name3 has no strategy
	name3, _ := ndn.NameFromString("/test/name_second/202=abc123")
	FibStrategyTable.InsertNextHop(name3, hopId3, 40)
	FibStrategyTable.InsertNextHop(name3, hopId3, 50)

	fse := FibStrategyTable.GetAllFIBEntries()
	assert.Equal(t, 2, len(fse))

	sort.Slice(fse, func(i, j int) bool {
		// Sort by name
		return fse[i].Name().String() < fse[j].Name().String()
	})

	assert.True(t, name2.Equals(fse[0].Name()))
	assert.True(t, bestRoute.Equals(fse[0].GetStrategy()))
	nextHops := fse[0].GetNextHops()
	assert.Equal(t, 2, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, uint64(20), nextHops[0].Cost)
	assert.Equal(t, hopId3, nextHops[1].Nexthop)
	assert.Equal(t, uint64(30), nextHops[1].Cost)

	// Only next hops, no strategy
	assert.True(t, name3.Equals(fse[1].Name()))
	assert.Nil(t, fse[1].GetStrategy())
	nextHops = fse[1].GetNextHops()
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId3, nextHops[0].Nexthop)
	assert.Equal(t, uint64(50), nextHops[0].Cost)
}

func TestGetAllForwardingStrategies_HT(t *testing.T) {
	newFibStrategyTableHashTable(1)
	assert.NotNil(t, FibStrategyTable)

	bestRoute, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := ndn.NameFromString("/localhost/nfd/strategy/multicast/v=1")

	hopId2 := uint64(200)
	hopId3 := uint64(300)

	// No strategy, so it shouldn't be included
	name, _ := ndn.NameFromString("/test")
	FibStrategyTable.InsertNextHop(name, hopId2, 20)

	name2, _ := ndn.NameFromString("/test/name/202=abc123")
	FibStrategyTable.SetStrategy(name2, bestRoute)
	FibStrategyTable.InsertNextHop(name2, hopId2, 20)
	FibStrategyTable.InsertNextHop(name2, hopId3, 30)

	name3, _ := ndn.NameFromString("/test/name_second/202=abc123")
	FibStrategyTable.SetStrategy(name3, multicast)

	fse := FibStrategyTable.GetAllForwardingStrategies()
	// Here, the "/" has a default strategy, bestRoute in this case
	assert.Equal(t, 3, len(fse))

	sort.Slice(fse, func(i, j int) bool {
		// Sort by name
		return fse[i].Name().String() < fse[j].Name().String()
	})

	rootName, _ := ndn.NameFromString("/")
	assert.True(t, rootName.Equals(fse[0].Name()))
	assert.True(t, bestRoute.Equals(fse[0].GetStrategy()))

	assert.True(t, name2.Equals(fse[1].Name()))
	assert.True(t, bestRoute.Equals(fse[1].GetStrategy()))
	nextHops := fse[1].GetNextHops()
	assert.Equal(t, 2, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, uint64(20), nextHops[0].Cost)
	assert.Equal(t, hopId3, nextHops[1].Nexthop)
	assert.Equal(t, uint64(30), nextHops[1].Cost)

	assert.True(t, name3.Equals(fse[2].Name()))
	assert.True(t, multicast.Equals(fse[2].GetStrategy()))
	nextHops = fse[2].GetNextHops()
	assert.Equal(t, 0, len(nextHops))
}

func testFIB_HT_Details(t *testing.T, m uint16) {
	// A test suite specific to the hash table approach
	newFibStrategyTableHashTable(m)
	assert.NotNil(t, FibStrategyTable)

	name1, _ := ndn.NameFromString("/a")
	name2, _ := ndn.NameFromString("/a/b/d")
	name3, _ := ndn.NameFromString("/a/b/c")
	name4, _ := ndn.NameFromString("/a/b/d/e")
	name5, _ := ndn.NameFromString("/a/b/d/e/f")
	nameNonexistent, _ := ndn.NameFromString(("/asdfasdf"))

	hopId1 := uint64(10)
	hopId2 := uint64(11)
	hopId3 := uint64(12)
	hopId4a := uint64(13)
	hopId4b := uint64(14)
	hopId5 := uint64(15)

	cost := uint64(1)

	FibStrategyTable.InsertNextHop(name1, hopId1, cost)
	FibStrategyTable.InsertNextHop(name2, hopId2, cost)
	FibStrategyTable.InsertNextHop(name3, hopId3, cost)
	FibStrategyTable.InsertNextHop(name4, hopId4a, cost)
	FibStrategyTable.InsertNextHop(name4, hopId4b, cost)
	FibStrategyTable.InsertNextHop(name5, hopId5, cost)

	nextHopsEmpty := FibStrategyTable.FindNextHops(nameNonexistent)
	assert.Equal(t, 0, len(nextHopsEmpty))

	nameSearch1, _ := ndn.NameFromString("/a/b/c/d/e/f")
	// match /a/b/c
	nextHops1 := FibStrategyTable.FindNextHops(nameSearch1)
	assert.Equal(t, 1, len(nextHops1))
	assert.Equal(t, hopId3, nextHops1[0].Nexthop)
	assert.Equal(t, cost, nextHops1[0].Cost)

	nameSearch2, _ := ndn.NameFromString("/a/c/d/e/f/g")
	// match /a
	nextHops2 := FibStrategyTable.FindNextHops(nameSearch2)
	assert.Equal(t, 1, len(nextHops2))
	assert.Equal(t, hopId1, nextHops2[0].Nexthop)
	assert.Equal(t, cost, nextHops2[0].Cost)

	nextHops3 := FibStrategyTable.FindNextHops(name1)
	// match /a
	assert.Equal(t, 1, len(nextHops3))
	assert.Equal(t, hopId1, nextHops3[0].Nexthop)
	assert.Equal(t, cost, nextHops3[0].Cost)

	nameSearch3, _ := ndn.NameFromString("/a/b/d/e/zzz")
	// match /a/b/d/e
	nextHops4 := FibStrategyTable.FindNextHops(nameSearch3)
	assert.Equal(t, 2, len(nextHops4))
	assert.Equal(t, hopId4a, nextHops4[0].Nexthop)
	assert.Equal(t, cost, nextHops4[0].Cost)
	assert.Equal(t, hopId4b, nextHops4[1].Nexthop)
	assert.Equal(t, cost, nextHops4[1].Cost)

	nameSearch4, _ := ndn.NameFromString("/a/b/d/e/f/g/h")
	// match /a/b/d/e/f
	nextHops5 := FibStrategyTable.FindNextHops(nameSearch4)
	assert.Equal(t, 1, len(nextHops5))
	assert.Equal(t, hopId5, nextHops5[0].Nexthop)
	assert.Equal(t, cost, nextHops5[0].Cost)

	nameSearch5, _ := ndn.NameFromString("/b")
	// match nothing
	nextHops6 := FibStrategyTable.FindNextHops(nameSearch5)
	assert.Equal(t, 0, len(nextHops6))

	nameSearch6, _ := ndn.NameFromString("/z/y/x/w/v/u/t")
	// match nothing
	nextHops7 := FibStrategyTable.FindNextHops(nameSearch6)
	assert.Equal(t, 0, len(nextHops7))

	// Deletions
	// Delete nonexistent name or face
	FibStrategyTable.RemoveNextHop(nameNonexistent, uint64(0))
	FibStrategyTable.RemoveNextHop(name1, uint64(0))

	// Delete with name and outFace that exists
	FibStrategyTable.RemoveNextHop(name4, hopId4b)
	nextHops := FibStrategyTable.FindNextHops(name4)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId4a, nextHops4[0].Nexthop)
	assert.Equal(t, cost, nextHops4[0].Cost)

	FibStrategyTable.RemoveNextHop(name4, hopId4a)
	// This should only match name2 now: /a/b/d
	nextHops = FibStrategyTable.FindNextHops(name4)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, cost, nextHops[0].Cost)

	FibStrategyTable.RemoveNextHop(name3, hopId3)
	// This should only match name1 now: /a
	nextHops = FibStrategyTable.FindNextHops(name3)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId1, nextHops[0].Nexthop)
	assert.Equal(t, cost, nextHops[0].Cost)

	// This should trigger pruning
	FibStrategyTable.RemoveNextHop(name5, hopId5)
	nextHops = FibStrategyTable.FindNextHops(name5)
	// This should only match name2 now: /a/b/d
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, cost, nextHops[0].Cost)

	FibStrategyTable.RemoveNextHop(name1, hopId1)
	nextHops = FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nextHops))

	nextHops = FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, cost, nextHops[0].Cost)

	FibStrategyTable.RemoveNextHop(name2, hopId2)

	// No entries left in FIB, everything should be removed
	nextHops = FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops(name3)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops(name4)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops(name5)
	assert.Equal(t, 0, len(nextHops))

	// Eliminate the root name from tree too
	// This results in an empty hash table
	rootName, _ := ndn.NameFromString("/")
	FibStrategyTable.UnsetStrategy(rootName)
	nextHops = FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops(name5)
	assert.Equal(t, 0, len(nextHops))
	strategy := FibStrategyTable.FindStrategy(name1)
	assert.Nil(t, strategy)

	// Nexthop entry exists but not root strategy
	FibStrategyTable.InsertNextHop(name1, hopId1, cost)
	strategy = FibStrategyTable.FindStrategy(name1)
	assert.Nil(t, strategy)
}
func TestFIB_HT_RealAndVirtualNodes(t *testing.T) {
	// Hashtable specific tests, for different values of m
	for i := 1; i < 8; i++ {
		testFIB_HT_Details(t, uint16(i))
	}
}
