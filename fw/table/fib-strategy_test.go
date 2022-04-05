/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table_test

import (
	"testing"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"

	"github.com/stretchr/testify/assert"
)

func TestNexthops(t *testing.T) {
	assert.NotNil(t, table.FibStrategyTable)

	name1, _ := ndn.NameFromString("/")
	nexthops1 := table.FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nexthops1))

	name2, _ := ndn.NameFromString("/test")
	nexthops2 := table.FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 0, len(nexthops2))
	table.FibStrategyTable.InsertNextHop(name2, 25, 1)
	table.FibStrategyTable.InsertNextHop(name2, 101, 10)
	nexthops2a := table.FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 2, len(nexthops2a))
	assert.Equal(t, uint64(25), nexthops2a[0].Nexthop)
	assert.Equal(t, uint64(1), nexthops2a[0].Cost)
	assert.Equal(t, uint64(101), nexthops2a[1].Nexthop)
	assert.Equal(t, uint64(10), nexthops2a[1].Cost)

	name3, _ := ndn.NameFromString("/test/name/202=abc123")
	nexthops3 := table.FibStrategyTable.FindNextHops(name3)
	assert.Equal(t, 2, len(nexthops3))
	assert.Equal(t, uint64(25), nexthops3[0].Nexthop)
	assert.Equal(t, uint64(1), nexthops3[0].Cost)
	assert.Equal(t, uint64(101), nexthops3[1].Nexthop)
	assert.Equal(t, uint64(10), nexthops3[1].Cost)
	nexthops1a := table.FibStrategyTable.FindNextHops(name1)
	assert.Equal(t, 0, len(nexthops1a))

	table.FibStrategyTable.RemoveNextHop(name2, 25)
	nexthops2b := table.FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 1, len(nexthops2b))
	assert.Equal(t, uint64(101), nexthops2b[0].Nexthop)
	assert.Equal(t, uint64(10), nexthops2b[0].Cost)

	// Test pruning
	table.FibStrategyTable.RemoveNextHop(name2, 101)
	nexthops2c := table.FibStrategyTable.FindNextHops(name2)
	assert.Equal(t, 0, len(nexthops2c))
}

func TestStrategies(t *testing.T) {
	assert.NotNil(t, table.FibStrategyTable)

	bestRoute, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := ndn.NameFromString("/localhost/nfd/strategy/multicast/v=1")

	name1, _ := ndn.NameFromString("/")
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.FindStrategy(name1)))

	name2, _ := ndn.NameFromString("/test")
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.FindStrategy(name2)))
	table.FibStrategyTable.SetStrategy(name2, multicast)
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name2)))

	name3, _ := ndn.NameFromString("/test/name/202=abc123")
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name3)))
	table.FibStrategyTable.SetStrategy(name3, bestRoute)
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name2)))
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.FindStrategy(name3)))

	// Test pruning
	table.FibStrategyTable.UnsetStrategy(name3)
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name2)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name3)))

	table.FibStrategyTable.SetStrategy(name1, multicast)
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name2)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.FindStrategy(name3)))
}
