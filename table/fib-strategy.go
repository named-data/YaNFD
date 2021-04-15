/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"container/list"
	"sync"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
)

// FibStrategyEntry represents an entry in the FIB-Strategy table.
type FibStrategyEntry struct {
	component ndn.NameComponent
	Name      *ndn.Name
	depth     int

	parent   *FibStrategyEntry
	children []*FibStrategyEntry

	nexthops []*FibNextHopEntry
	strategy *ndn.Name
}

// FibNextHopEntry represents a nexthop in a FIB entry.
type FibNextHopEntry struct {
	Nexthop uint64
	Cost    uint64
}

// FibStrategyTable is a table containing FIB and Strategy entries for given prefixes.
var FibStrategyTable *FibStrategyEntry
var fibStrategyRWMutex sync.RWMutex
var fibPrefixes map[string]*FibStrategyEntry

func init() {
	var err error
	FibStrategyTable = new(FibStrategyEntry)
	FibStrategyTable.component = nil // Root component will be nil since it represents zero components
	FibStrategyTable.strategy, err = ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	if err != nil {
		core.LogFatal("FibStrategy", "Unable to create strategy name for best-route for \"/\": ", err)
	}
	FibStrategyTable.Name = ndn.NewName()
	fibPrefixes = make(map[string]*FibStrategyEntry)
}

func (f *FibStrategyEntry) findExactMatchEntry(name *ndn.Name) *FibStrategyEntry {
	if name.Size() > f.depth {
		for _, child := range f.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findExactMatchEntry(name)
			}
		}
	} else if name.Size() == f.depth {
		return f
	}
	return nil
}

func (f *FibStrategyEntry) findLongestPrefixEntry(name *ndn.Name) *FibStrategyEntry {
	if name.Size() > f.depth {
		for _, child := range f.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findLongestPrefixEntry(name)
			}
		}
	}
	return f
}

func (f *FibStrategyEntry) fillTreeToPrefix(name *ndn.Name) *FibStrategyEntry {
	curNode := f.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(FibStrategyEntry)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		curNode.children = append(curNode.children, newNode)
		curNode = newNode
	}
	return curNode
}

func (f *FibStrategyEntry) pruneIfEmpty() {
	for curNode := f; curNode.parent != nil && len(curNode.children) == 0 && len(curNode.nexthops) == 0 && curNode.strategy == nil; curNode = curNode.parent {
		// Remove from parent's children
		for i, child := range curNode.parent.children {
			if child == f {
				if i < len(curNode.parent.children)-1 {
					copy(curNode.parent.children[i:], curNode.parent.children[i+1:])
				}
				curNode.parent.children = curNode.parent.children[:len(curNode.parent.children)-1]
				break
			}
		}
	}
}

// LongestPrefixNexthops returns the longest-prefix matching nexthop(s) matching the specified name.
func (f *FibStrategyEntry) LongestPrefixNexthops(name *ndn.Name) []*FibNextHopEntry {
	fibStrategyRWMutex.RLock()

	// Find longest prefix matching entry
	curNode := f.findLongestPrefixEntry(name)

	// Now step back up until we find a nexthop entry
	var nexthops []*FibNextHopEntry
	for ; curNode != nil; curNode = curNode.parent {
		if len(curNode.nexthops) > 0 {
			nexthops = make([]*FibNextHopEntry, len(curNode.nexthops))
			for i, nexthop := range curNode.nexthops {
				nexthops[i] = nexthop
			}
			break
		}
	}

	fibStrategyRWMutex.RUnlock()
	return nexthops
}

// LongestPrefixStrategy returns the longest-prefix matching strategy choice entry for the specified name.
func (f *FibStrategyEntry) LongestPrefixStrategy(name *ndn.Name) *ndn.Name {
	fibStrategyRWMutex.RLock()

	// Find longest prefix matching entry
	curNode := f.findLongestPrefixEntry(name)

	// Now step back up until we find a nexthop entry
	var strategy *ndn.Name
	for ; curNode != nil; curNode = curNode.parent {
		if curNode.strategy != nil {
			strategy = curNode.strategy
			break
		}
	}

	fibStrategyRWMutex.RUnlock()
	return strategy
}

// AddNexthop adds or updates a nexthop entry for the specified prefix.
func (f *FibStrategyEntry) AddNexthop(name *ndn.Name, nexthop uint64, cost uint64) {
	fibStrategyRWMutex.Lock()
	entry := f.fillTreeToPrefix(name)
	if entry.Name == nil {
		entry.Name = name
	}
	for _, existingNexthop := range entry.nexthops {
		if existingNexthop.Nexthop == nexthop {
			existingNexthop.Cost = cost
			fibStrategyRWMutex.Unlock()
			return
		}
	}

	newEntry := new(FibNextHopEntry)
	newEntry.Nexthop = nexthop
	newEntry.Cost = cost
	entry.nexthops = append(entry.nexthops, newEntry)
	fibPrefixes[name.String()] = entry
	fibStrategyRWMutex.Unlock()
}

// ClearNexthops clears all nexthops for the specified prefix.
func (f *FibStrategyEntry) ClearNexthops(name *ndn.Name) {
	fibStrategyRWMutex.Lock()
	node := f.findExactMatchEntry(name)
	if node != nil {
		node.nexthops = make([]*FibNextHopEntry, 0)
	}
	fibStrategyRWMutex.Unlock()
}

// GetNexthops gets nexthops in the specified entry.
func (f *FibStrategyEntry) GetNexthops() []*FibNextHopEntry {
	return f.nexthops
}

// RemoveNexthop removes the specified nexthop entry from the specified prefix.
func (f *FibStrategyEntry) RemoveNexthop(name *ndn.Name, nexthop uint64) {
	fibStrategyRWMutex.Lock()
	entry := f.findExactMatchEntry(name)
	if entry != nil {
		for i, existingNexthop := range entry.nexthops {
			if existingNexthop.Nexthop == nexthop {
				if i < len(entry.nexthops)-1 {
					copy(entry.nexthops[i:], entry.nexthops[i+1:])
				}
				entry.nexthops = entry.nexthops[:len(entry.nexthops)-1]
				break
			}
		}
		if len(entry.nexthops) == 0 {
			delete(fibPrefixes, name.String())
		}
		entry.pruneIfEmpty()
	}
	fibStrategyRWMutex.Unlock()
}

// GetAllFIBEntries returns all nexthop entries in the FIB.
func (f *FibStrategyEntry) GetAllFIBEntries() []*FibStrategyEntry {
	fibStrategyRWMutex.RLock()
	entries := make([]*FibStrategyEntry, 0)
	// Walk tree in-order
	queue := list.New()
	queue.PushBack(f)
	for queue.Len() > 0 {
		fsEntry := queue.Front().Value.(*FibStrategyEntry)
		queue.Remove(queue.Front())
		// Add all children to stack
		for _, child := range fsEntry.children {
			queue.PushFront(child)
		}

		// If has any nexthop entries, add to list
		if len(fsEntry.nexthops) > 0 {
			entries = append(entries, fsEntry)
		}
	}
	fibStrategyRWMutex.RUnlock()
	return entries
}

// SetStrategy sets the strategy for the specified prefix.
func (f *FibStrategyEntry) SetStrategy(name *ndn.Name, strategy *ndn.Name) {
	fibStrategyRWMutex.Lock()
	entry := f.fillTreeToPrefix(name)
	if entry.Name == nil {
		entry.Name = name
	}
	entry.strategy = strategy
	fibStrategyRWMutex.Unlock()
}

// UnsetStrategy unsets the strategy for the specified prefix.
func (f *FibStrategyEntry) UnsetStrategy(name *ndn.Name) {
	fibStrategyRWMutex.Lock()
	entry := f.findExactMatchEntry(name)
	if entry != nil {
		entry.strategy = nil
		entry.pruneIfEmpty()
	}
	fibStrategyRWMutex.Unlock()
}

// GetStrategy gets the strategy set at the current node.
func (f *FibStrategyEntry) GetStrategy() *ndn.Name {
	return f.strategy
}

// GetAllStrategyChoices returns all strategy choice entries in the Strategy Table.
func (f *FibStrategyEntry) GetAllStrategyChoices() []*FibStrategyEntry {
	fibStrategyRWMutex.RLock()
	entries := make([]*FibStrategyEntry, 0)
	// Walk tree in-order
	queue := list.New()
	queue.PushBack(f)
	for queue.Len() > 0 {
		fsEntry := queue.Front().Value.(*FibStrategyEntry)
		queue.Remove(queue.Front())
		// Add all children to stack
		for _, child := range fsEntry.children {
			queue.PushFront(child)
		}

		// If has any nexthop entries, add to list
		if fsEntry.strategy != nil {
			entries = append(entries, fsEntry)
		}
	}
	fibStrategyRWMutex.RUnlock()
	return entries
}
