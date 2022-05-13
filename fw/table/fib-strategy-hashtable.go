/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2022 Danning Yu.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

// This file implements a hash table version of the FIB, as outlined in section 2.3 of:
// W. So, A. Narayanan and D. Oran,
// "Named data networking on a router: Fast and DoS-resistant forwarding with hash tables,"
// Architectures for Networking and Communications Systems, 2013, pp. 215-225,
// doi: 10.1109/ANCS.2013.6665203.

package table

import (
	"sync"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/utils/comparison"
)

type virtualDetails struct {
	// md is the max depth associated with this virtual node, as defined in the paper.
	md int
}

// FibStrategyHashTable represents a tree implementation of the FIB-Strategy table.
type FibStrategyHashTable struct {
	// m is the name length for virtual nodes, as defined in the paper
	// Must be a positive value
	m int

	// realTable is a map of names (hashed as uint64 values) to the FIB entry
	// associated with that name.
	realTable map[string]*baseFibStrategyEntry

	// virtTable is a map of virtual names (hashed as uint64 values) to the
	// virtualDetails struct associated with that virtual name.
	virtTable map[string]*virtualDetails

	// virtTableNames is a map of virtual names (hashed as uint64 values) to
	// a set of all the real names associated with that virtual name. The
	// inner map is being used as a set; the bool value type does not matter.
	virtTableNames map[string](map[string]struct{})

	// fibStrategyRWMutex is a mutex used to synchronize accesses to the FIB,
	// which is shared across all the forwarding threads.
	fibStrategyRWMutex sync.RWMutex
}

// newFibStrategyTableHashTable creates a new FIB with the hash table algorithm.
// The argument m determines the virtual name length.
func newFibStrategyTableHashTable(m uint16) {
	FibStrategyTable = new(FibStrategyHashTable)
	fibStrategyTableHashTable := FibStrategyTable.(*FibStrategyHashTable)

	fibStrategyTableHashTable.m = int(m) // Cast to int so that it's easy to pass to name.Prefix
	fibStrategyTableHashTable.realTable = make(map[string]*baseFibStrategyEntry)
	fibStrategyTableHashTable.virtTable = make(map[string]*virtualDetails)
	fibStrategyTableHashTable.virtTableNames = make(map[string]map[string]struct{})

	rootName, _ := ndn.NameFromString(("/"))
	defaultStrategy, _ := ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")

	rtEntry := new(baseFibStrategyEntry)
	rtEntry.name = rootName
	rtEntry.strategy = defaultStrategy
	fibStrategyTableHashTable.realTable[rootName.String()] = rtEntry
}

// findLongestPrefixMatch returns the entry corresponding to the longest
// prefix match of the given name. It returns nil if no exact match was found.
func (f *FibStrategyHashTable) findLongestPrefixMatch(name *ndn.Name) *baseFibStrategyEntry {
	if name.Size() <= f.m {
		// Name length is less than or equal to M, so only need to check real table
		for pfx := name.Size(); pfx >= 0; pfx-- {
			if val, ok := f.realTable[name.Prefix(pfx).String()]; ok {
				return val
			}
		}
		return nil
	}

	// Name is longer than M, so use virtual node to lookup first
	virtName := name.Prefix(f.m)
	_, ok := f.virtTable[virtName.String()]
	if ok {
		// Virtual name present, look for longer matches
		pfx := comparison.Min(f.virtTable[virtName.String()].md, name.Size())
		for ; pfx > f.m; pfx-- {
			if val, ok := f.realTable[name.Prefix(pfx).String()]; ok {
				return val
			}
		}
	}

	// Name is longer that M but did not find a match for names with length > M
	// Start looking in the real table from length M
	// For example: Table has prefixes /a and /a/b/c, virtual entry is /a/b
	// A search for /a/b/d will not match /a/b/c, so we need it to match /a
	for pfx := f.m; pfx >= 0; pfx-- {
		if val, ok := f.realTable[name.Prefix(pfx).String()]; ok {
			return val
		}
	}

	// no match found
	return nil
}

// insertEntry inserts an entry into the FIB table, specifically realTable,
// virtTable, and virtTableNames. It does not set the nexthops or strategy
// fields of the newly created entry. The caller is responsible for doing that.
func (f *FibStrategyHashTable) insertEntry(name *ndn.Name) *baseFibStrategyEntry {
	nameString := name.String()

	if _, ok := f.realTable[nameString]; !ok {
		rtEntry := new(baseFibStrategyEntry)
		rtEntry.name = name
		f.realTable[nameString] = rtEntry
	}

	// Insert into virtual table if name size >= M
	if name.Size() == f.m {
		if _, ok := f.virtTable[nameString]; !ok {
			vtEntry := new(virtualDetails)
			vtEntry.md = name.Size()
			f.virtTable[nameString] = vtEntry
		}

		if _, ok := f.virtTableNames[nameString]; !ok {
			f.virtTableNames[nameString] = make(map[string]struct{})
		}

		f.virtTable[nameString].md = comparison.Max(f.virtTable[nameString].md, name.Size())

		// Insert into set of names
		f.virtTableNames[nameString][nameString] = struct{}{}

	} else if name.Size() > f.m {
		virtNameString := name.Prefix(f.m).String()
		if _, ok := f.virtTable[virtNameString]; !ok {
			vtEntry := new(virtualDetails)
			vtEntry.md = name.Size()
			f.virtTable[virtNameString] = vtEntry
		}

		if _, ok := f.virtTableNames[virtNameString]; !ok {
			f.virtTableNames[virtNameString] = make(map[string]struct{})
		}

		f.virtTable[virtNameString].md = comparison.Max(f.virtTable[virtNameString].md, name.Size())

		// Insert into set of names
		f.virtTableNames[virtNameString][nameString] = struct{}{}
	}

	return f.realTable[nameString]
}

// pruneTables takes in an entry and removes it from the real table if it has no
// next hops and no strategy associated with it. It also eliminates its
// corresponding virtual entry, if applicable.
func (f *FibStrategyHashTable) pruneTables(entry *baseFibStrategyEntry) {
	var pruned bool
	name := entry.name

	// Delete the real entry
	if len(entry.nexthops) == 0 && entry.strategy == nil {
		delete(f.realTable, entry.name.String())
		pruned = true
	}

	if !pruned {
		return
	}

	// Delete the virtual entry too, if needed
	if name.Size() >= f.m {
		virtNameString := name.Prefix(f.m).String()
		virtEntry, inVirtTable := f.virtTable[virtNameString]
		virtTableNamesEntry, inVirtNameTable := f.virtTableNames[virtNameString]
		namePresentForVirtName := false
		if inVirtNameTable {
			_, namePresentForVirtName = virtTableNamesEntry[name.String()]
		}

		// If virtual name is present in table
		// AND real name is associated with this virtual name
		// AND this real name was deleted from the real table
		// Then delete from virtualTableNames if it's last real name
		// associated with the virtual name
		if inVirtTable && namePresentForVirtName && pruned {
			delete(virtTableNamesEntry, name.String())
			if len(virtTableNamesEntry) == 0 {
				delete(f.virtTableNames, virtNameString)
			}
		}

		// If virtual name is present in table
		// AND the real name being deleted was the longest associated with the virtual name
		// AND this real name was deleted from the real table
		if inVirtTable && name.Size() == virtEntry.md && pruned {
			_, inVirtNameTable = f.virtTableNames[virtNameString]
			if !inVirtNameTable {
				// Delete the entry entirely from the virtual table too
				// if it was removed it from the virtual name table
				delete(f.virtTable, virtNameString)
			} else {
				// Update with length of next longest real prefix associated
				// with this virtual prefix
				for k, _ := range f.virtTableNames[virtNameString] {
					n, _ := ndn.NameFromString(k)
					virtEntry.md = comparison.Max(virtEntry.md, n.Size())
				}
			}
		}
	}
}

// FindNextHops returns the longest-prefix matching nexthop(s) matching the specified name.
func (f *FibStrategyHashTable) FindNextHops(name *ndn.Name) []*FibNextHopEntry {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()

	entry := f.findLongestPrefixMatch(name)

	if entry == nil {
		return nil
	}

	// Go backwards to find the first entry with nexthops
	// since some might only have a strategy but no nexthops
	for pfx := entry.name.Size(); pfx >= 0; pfx-- {
		val, ok := f.realTable[entry.name.Prefix(pfx).String()]
		if ok && len(val.nexthops) > 0 {
			return val.nexthops
		}
	}

	return nil
}

// FindStrategy returns the longest-prefix matching strategy choice entry for the specified name.
func (f *FibStrategyHashTable) FindStrategy(name *ndn.Name) *ndn.Name {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()

	entry := f.findLongestPrefixMatch(name)

	if entry == nil {
		return nil
	}

	// Go backwards to find the first entry with strategy
	// since some might only have a nexthops but no strategy
	for pfx := entry.name.Size(); pfx >= 0; pfx-- {
		val, ok := f.realTable[entry.name.Prefix(pfx).String()]
		if ok && val.strategy != nil {
			return val.strategy
		}
	}

	return nil
}

// InsertNextHop adds or updates a nexthop entry for the specified prefix.
func (f *FibStrategyHashTable) InsertNextHop(name *ndn.Name, nexthop uint64, cost uint64) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()

	realEntry := f.insertEntry(name)

	for _, existingNextHop := range realEntry.nexthops {
		if existingNextHop.Nexthop == nexthop {
			// Update existing hop
			existingNextHop.Cost = cost
			return
		}
	}

	// Did not update existing hop, so add a new hop
	nhEntry := new(FibNextHopEntry)
	nhEntry.Nexthop = nexthop
	nhEntry.Cost = cost
	realEntry.nexthops = append(realEntry.nexthops, nhEntry)

}

// ClearNextHops clears all nexthops for the specified prefix.
func (f *FibStrategyHashTable) ClearNextHops(name *ndn.Name) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()

	entry, ok := f.realTable[name.String()]
	if ok {
		entry.nexthops = make([]*FibNextHopEntry, 0)
		f.pruneTables(entry)
	}
}

// RemoveNextHop removes the specified nexthop entry from the specified prefix.
func (f *FibStrategyHashTable) RemoveNextHop(name *ndn.Name, nexthop uint64) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()

	if _, ok := f.realTable[name.String()]; !ok {
		return
	}

	// Remove matching nexthop from real table (if one exists)
	realEntry := f.realTable[name.String()]
	nextHops := realEntry.nexthops
	for i, nh := range nextHops {
		if nh.Nexthop == nexthop {
			if len(nextHops) > 1 {
				nextHops[i] = nextHops[len(nextHops)-1]
			}
			realEntry.nexthops = nextHops[:len(nextHops)-1]
			f.pruneTables(realEntry)
			break
		}
	}
}

// GetAllFIBEntries returns all nexthop entries in the FIB.
func (f *FibStrategyHashTable) GetAllFIBEntries() []FibStrategyEntry {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()
	entries := make([]FibStrategyEntry, 0)
	for _, v := range f.realTable {
		if len(v.nexthops) > 0 {
			entries = append(entries, v)
		}
	}

	return entries
}

// SetStrategy sets the strategy for the specified prefix.
func (f *FibStrategyHashTable) SetStrategy(name *ndn.Name, strategy *ndn.Name) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()

	realEntry := f.insertEntry(name)
	realEntry.strategy = strategy
}

// UnsetStrategy unsets the strategy for the specified prefix.
func (f *FibStrategyHashTable) UnsetStrategy(name *ndn.Name) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()

	entry, ok := f.realTable[name.String()]
	if ok {
		entry.strategy = nil
		f.pruneTables(entry)
	}
}

// GetAllForwardingStrategies returns all strategy choice entries in the Strategy Table.
func (f *FibStrategyHashTable) GetAllForwardingStrategies() []FibStrategyEntry {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()
	entries := make([]FibStrategyEntry, 0)
	for _, v := range f.realTable {
		if v.strategy != nil {
			entries = append(entries, v)
		}
	}

	return entries
}
