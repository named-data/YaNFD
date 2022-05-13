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

	"github.com/named-data/YaNFD/ndn"
)

type fibStrategyTreeEntry struct {
	baseFibStrategyEntry
	depth    int
	parent   *fibStrategyTreeEntry
	children []*fibStrategyTreeEntry
}

// FibStrategy Tree represents a tree implementation of the FIB-Strategy table.
type FibStrategyTree struct {
	root *fibStrategyTreeEntry

	// fibStrategyRWMutex is a mutex used to synchronize accesses to the FIB,
	// which is shared across all the forwarding threads.
	fibStrategyRWMutex sync.RWMutex

	fibPrefixes map[string]*fibStrategyTreeEntry
}

func newFibStrategyTableTree() {
	FibStrategyTable = new(FibStrategyTree)
	fibStrategyTableTree := FibStrategyTable.(*FibStrategyTree)
	fibStrategyTableTree.root = new(fibStrategyTreeEntry)
	fibStrategyTableTree.root.component = nil // Root component will be nil since it represents zero components
	fibStrategyTableTree.root.strategy, _ = ndn.NameFromString("/localhost/nfd/strategy/best-route/v=1")
	fibStrategyTableTree.root.name = ndn.NewName()
	fibStrategyTableTree.fibPrefixes = make(map[string]*fibStrategyTreeEntry)
}

// findExactMatchEntry returns the entry corresponding to the exact match of
// the given name. It returns nil if no exact match was found.
func (f *fibStrategyTreeEntry) findExactMatchEntry(name *ndn.Name) *fibStrategyTreeEntry {
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

// findLongestPrefixEntry returns the entry corresponding to the longest
// prefix match of the given name. It returns nil if no exact match was found.
func (f *fibStrategyTreeEntry) findLongestPrefixEntry(name *ndn.Name) *fibStrategyTreeEntry {
	if name.Size() > f.depth {
		for _, child := range f.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findLongestPrefixEntry(name)
			}
		}
	}
	return f
}

// fillTreeToPrefix breaks the given name into components and adds nodes to the
// tree for any missing components.
func (f *FibStrategyTree) fillTreeToPrefix(name *ndn.Name) *fibStrategyTreeEntry {
	curNode := f.root.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(fibStrategyTreeEntry)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		curNode.children = append(curNode.children, newNode)
		curNode = newNode
	}
	return curNode
}

// pruneIfEmpty prunes nodes from the tree if they no longer carry any information,
// where information is the combination of child nodes, nexthops, and strategies.
func (f *fibStrategyTreeEntry) pruneIfEmpty() {
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

// FindNextHops returns the longest-prefix matching nexthop(s) matching the specified name.
func (f *FibStrategyTree) FindNextHops(name *ndn.Name) []*FibNextHopEntry {
	f.fibStrategyRWMutex.RLock()

	// Find longest prefix matching entry
	curNode := f.root.findLongestPrefixEntry(name)

	// Now step back up until we find a nexthops entry
	// since some might only have a strategy but no nexthops
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

	f.fibStrategyRWMutex.RUnlock()
	return nexthops
}

// FindStrategy returns the longest-prefix matching strategy choice entry for the specified name.
func (f *FibStrategyTree) FindStrategy(name *ndn.Name) *ndn.Name {
	f.fibStrategyRWMutex.RLock()

	// Find longest prefix matching entry
	curNode := f.root.findLongestPrefixEntry(name)

	// Now step back up until we find a strategy entry
	// since some might only have a nexthops but no strategy
	var strategy *ndn.Name
	for ; curNode != nil; curNode = curNode.parent {
		if curNode.strategy != nil {
			strategy = curNode.strategy
			break
		}
	}

	f.fibStrategyRWMutex.RUnlock()
	return strategy
}

// InsertNextHop adds or updates a nexthop entry for the specified prefix.
func (f *FibStrategyTree) InsertNextHop(name *ndn.Name, nexthop uint64, cost uint64) {
	f.fibStrategyRWMutex.Lock()
	entry := f.fillTreeToPrefix(name)
	if entry.name == nil {
		entry.name = name
	}
	for _, existingNexthop := range entry.nexthops {
		if existingNexthop.Nexthop == nexthop {
			existingNexthop.Cost = cost
			f.fibStrategyRWMutex.Unlock()
			return
		}
	}

	newEntry := new(FibNextHopEntry)
	newEntry.Nexthop = nexthop
	newEntry.Cost = cost
	entry.nexthops = append(entry.nexthops, newEntry)
	f.fibPrefixes[name.String()] = entry
	f.fibStrategyRWMutex.Unlock()
}

// ClearNextHops clears all nexthops for the specified prefix.
func (f *FibStrategyTree) ClearNextHops(name *ndn.Name) {
	f.fibStrategyRWMutex.Lock()
	node := f.root.findExactMatchEntry(name)
	if node != nil {
		node.nexthops = make([]*FibNextHopEntry, 0)
	}
	f.fibStrategyRWMutex.Unlock()
}

// RemoveNextHop removes the specified nexthop entry from the specified prefix.
func (f *FibStrategyTree) RemoveNextHop(name *ndn.Name, nexthop uint64) {
	f.fibStrategyRWMutex.Lock()
	entry := f.root.findExactMatchEntry(name)
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
			delete(f.fibPrefixes, name.String())
		}
		entry.pruneIfEmpty()
	}
	f.fibStrategyRWMutex.Unlock()
}

// GetAllFIBEntries returns all nexthop entries in the FIB.
func (f *FibStrategyTree) GetAllFIBEntries() []FibStrategyEntry {
	f.fibStrategyRWMutex.RLock()
	entries := make([]FibStrategyEntry, 0)
	// Walk tree in-order
	queue := list.New()
	queue.PushBack(f.root)
	for queue.Len() > 0 {
		fsEntry := queue.Front().Value.(*fibStrategyTreeEntry)
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
	f.fibStrategyRWMutex.RUnlock()
	return entries
}

// SetStrategy sets the strategy for the specified prefix.
func (f *FibStrategyTree) SetStrategy(name *ndn.Name, strategy *ndn.Name) {
	f.fibStrategyRWMutex.Lock()
	entry := f.fillTreeToPrefix(name)
	if entry.name == nil {
		entry.name = name
	}
	entry.strategy = strategy
	f.fibStrategyRWMutex.Unlock()
}

// UnsetStrategy unsets the strategy for the specified prefix.
func (f *FibStrategyTree) UnsetStrategy(name *ndn.Name) {
	f.fibStrategyRWMutex.Lock()
	entry := f.root.findExactMatchEntry(name)
	if entry != nil {
		entry.strategy = nil
		entry.pruneIfEmpty()
	}
	f.fibStrategyRWMutex.Unlock()
}

// GetAllForwardingStrategies returns all strategy choice entries in the Strategy Table.
func (f *FibStrategyTree) GetAllForwardingStrategies() []FibStrategyEntry {
	f.fibStrategyRWMutex.RLock()
	entries := make([]FibStrategyEntry, 0)
	// Walk tree in-order
	queue := list.New()
	queue.PushBack(f.root)
	for queue.Len() > 0 {
		fsEntry := queue.Front().Value.(*fibStrategyTreeEntry)
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
	f.fibStrategyRWMutex.RUnlock()
	return entries
}
