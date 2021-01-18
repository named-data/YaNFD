/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table_test

import (
	"testing"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/table"

	"github.com/stretchr/testify/assert"
)

func TestNexthops(t *testing.T) {
	assert.NotNil(t, table.FibStrategyTable)

	name1, _ := ndn.NameFromString("/")
	nexthops1 := table.FibStrategyTable.LongestPrefixNexthops(name1)
	assert.Equal(t, 0, len(nexthops1))

	name2, _ := ndn.NameFromString("/test")
	nexthops2 := table.FibStrategyTable.LongestPrefixNexthops(name2)
	assert.Equal(t, 0, len(nexthops2))
	table.FibStrategyTable.AddNexthop(name2, 25, 1)
	table.FibStrategyTable.AddNexthop(name2, 101, 10)
	nexthops2a := table.FibStrategyTable.LongestPrefixNexthops(name2)
	assert.Equal(t, 2, len(nexthops2a))
	assert.Equal(t, 25, nexthops2a[0].Nexthop)
	assert.Equal(t, uint(1), nexthops2a[0].Cost)
	assert.Equal(t, 101, nexthops2a[1].Nexthop)
	assert.Equal(t, uint(10), nexthops2a[1].Cost)

	name3, _ := ndn.NameFromString("/test/name/202=abc123")
	nexthops3 := table.FibStrategyTable.LongestPrefixNexthops(name3)
	assert.Equal(t, 2, len(nexthops3))
	assert.Equal(t, 25, nexthops3[0].Nexthop)
	assert.Equal(t, uint(1), nexthops3[0].Cost)
	assert.Equal(t, 101, nexthops3[1].Nexthop)
	assert.Equal(t, uint(10), nexthops3[1].Cost)
	nexthops1a := table.FibStrategyTable.LongestPrefixNexthops(name1)
	assert.Equal(t, 0, len(nexthops1a))

	table.FibStrategyTable.RemoveNexthop(name2, 25)
	nexthops2b := table.FibStrategyTable.LongestPrefixNexthops(name2)
	assert.Equal(t, 1, len(nexthops2b))
	assert.Equal(t, 101, nexthops2b[0].Nexthop)
	assert.Equal(t, uint(10), nexthops2b[0].Cost)

	// Test pruning
	table.FibStrategyTable.RemoveNexthop(name2, 101)
	nexthops2c := table.FibStrategyTable.LongestPrefixNexthops(name2)
	assert.Equal(t, 0, len(nexthops2c))
}

func TestStrategies(t *testing.T) {
	assert.NotNil(t, table.FibStrategyTable)

	bestRoute, _ := ndn.NameFromString("/localhost/yanfd/strategy/best-route/%FD%01")
	multicast, _ := ndn.NameFromString("/localhost/yanfd/strategy/multicast/%FD%01")

	name1, _ := ndn.NameFromString("/")
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.LongestPrefixStrategy(name1)))

	name2, _ := ndn.NameFromString("/test")
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.LongestPrefixStrategy(name2)))
	table.FibStrategyTable.SetStrategy(name2, multicast)
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.LongestPrefixStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name2)))

	name3, _ := ndn.NameFromString("/test/name/202=abc123")
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name3)))
	table.FibStrategyTable.SetStrategy(name3, bestRoute)
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.LongestPrefixStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name2)))
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.LongestPrefixStrategy(name3)))

	// Test pruning
	table.FibStrategyTable.UnsetStrategy(name3)
	assert.True(t, bestRoute.Equals(table.FibStrategyTable.LongestPrefixStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name2)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name3)))

	table.FibStrategyTable.SetStrategy(name1, multicast)
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name1)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name2)))
	assert.True(t, multicast.Equals(table.FibStrategyTable.LongestPrefixStrategy(name3)))
}
