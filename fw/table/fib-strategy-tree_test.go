/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
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

func TestFindNextHops(t *testing.T) {
	newFibStrategyTable()

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

func TestFind_Set_Unset_Strategy(t *testing.T) {
	newFibStrategyTable()

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

func TestInsertNextHop(t *testing.T) {
	newFibStrategyTable()
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

func TestClearNextHops(t *testing.T) {
	newFibStrategyTable()
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

func TestRemoveNextHop(t *testing.T) {
	newFibStrategyTable()
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

func TestGetAllFIBEntries(t *testing.T) {
	newFibStrategyTable()
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

func TestGetAllForwardingStrategies(t *testing.T) {
	newFibStrategyTable()
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
