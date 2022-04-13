/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"github.com/named-data/YaNFD/ndn"
)

// FibStrategyEntry represents an entry in the FIB-Strategy table.
type FibStrategyEntry interface {
	Name() *ndn.Name
	GetStrategy() *ndn.Name
	GetNextHops() []*FibNextHopEntry
}

// baseFibStrategyEntry represents information that all
// FibStrategyEntry implementations should include.
type baseFibStrategyEntry struct {
	component ndn.NameComponent
	name      *ndn.Name
	nexthops  []*FibNextHopEntry
	strategy  *ndn.Name
}

// FibNextHopEntry represents a nexthop in a FIB entry.
type FibNextHopEntry struct {
	Nexthop uint64
	Cost    uint64
}

// FibStrategy represents the functionality that a FIB-strategy table should implement.
type FibStrategy interface {
	FindNextHops(name *ndn.Name) []*FibNextHopEntry
	FindStrategy(name *ndn.Name) *ndn.Name
	InsertNextHop(name *ndn.Name, nextHop uint64, cost uint64)
	ClearNextHops(name *ndn.Name)
	RemoveNextHop(name *ndn.Name, nextHop uint64)

	GetAllFIBEntries() []FibStrategyEntry

	SetStrategy(name *ndn.Name, strategy *ndn.Name)
	UnsetStrategy(name *ndn.Name)
	GetAllForwardingStrategies() []FibStrategyEntry
}

// FibStrategy is a table containing FIB and Strategy entries for given prefixes.
var FibStrategyTable FibStrategy

// Name returns the name associated with the baseFibStrategyEntry.
func (e *baseFibStrategyEntry) Name() *ndn.Name {
	return e.name
}

// GetStrategy returns the strategy associated with the baseFibStrategyEntry.
func (e *baseFibStrategyEntry) GetStrategy() *ndn.Name {
	return e.strategy
}

// GetNexthops gets the nexthops of the specified entry.
func (e *baseFibStrategyEntry) GetNextHops() []*FibNextHopEntry {
	return e.nexthops
}
