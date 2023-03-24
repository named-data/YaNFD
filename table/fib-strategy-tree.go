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

	"github.com/cespare/xxhash"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
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

	fibPrefixes map[uint64]*fibStrategyTreeEntry
}

func newFibStrategyTableTree() {
	FibStrategyTable = new(FibStrategyTree)
	fibStrategyTableTree := FibStrategyTable.(*FibStrategyTree)
	fibStrategyTableTree.root = new(fibStrategyTreeEntry)
	// Root component will be nil since it represents zero components
	fibStrategyTableTree.root.component = enc.Component{}
	base, _ := enc.NameFromStr("/localhost/nfd/strategy/best-route/v=1")
	fibStrategyTableTree.root.strategy = base
	fibStrategyTableTree.root.name = enc.Name{}
	fibStrategyTableTree.fibPrefixes = make(map[uint64]*fibStrategyTreeEntry)
}

// findExactMatchEntry returns the entry corresponding to the exact match of
// the given name. It returns nil if no exact match was found.

func (f *fibStrategyTreeEntry) findExactMatchEntryEnc(name enc.Name) *fibStrategyTreeEntry {
	if len(name) > f.depth {
		for _, child := range f.children {
			if At(name, child.depth-1).Equal(child.component) {
				return child.findExactMatchEntryEnc(name)
			}
		}
	} else if len(name) == f.depth {
		return f
	}
	return nil
}

// findLongestPrefixEntry returns the entry corresponding to the longest
// prefix match of the given name. It returns nil if no exact match was found.

func (f *fibStrategyTreeEntry) findLongestPrefixEntryEnc(name enc.Name) *fibStrategyTreeEntry {
	if len(name) > f.depth {
		for _, child := range f.children {
			if At(name, child.depth-1).Equal(child.component) {
				return child.findLongestPrefixEntryEnc(name)
			}
		}
	}
	return f
}

// fillTreeToPrefix breaks the given name into components and adds nodes to the
// tree for any missing components.

func (f *FibStrategyTree) fillTreeToPrefixEnc(name enc.Name) *fibStrategyTreeEntry {
	curNode := f.root.findLongestPrefixEntryEnc(name)
	for depth := curNode.depth + 1; depth <= len(name); depth++ {
		newNode := new(fibStrategyTreeEntry)
		newNode.component = At(name, depth-1)
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
func (f *fibStrategyTreeEntry) pruneIfEmptyEnc() {
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

func (f *FibStrategyTree) FindNextHopsEnc(name enc.Name) []*FibNextHopEntry {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()

	// Find longest prefix matching entry
	curNode := f.root.findLongestPrefixEntryEnc(name)

	// Now step back up until we find a nexthops entry
	// since some might only have a strategy but no nexthops
	var nexthops []*FibNextHopEntry
	for ; curNode != nil; curNode = curNode.parent {
		if len(curNode.nexthops) > 0 {
			nexthops = make([]*FibNextHopEntry, len(curNode.nexthops))
			copy(nexthops, curNode.nexthops)
			break
		}
	}

	return nexthops
}

// FindStrategy returns the longest-prefix matching strategy choice entry for the specified name.
func (f *FibStrategyTree) FindStrategyEnc(name enc.Name) enc.Name {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()

	// Find longest prefix matching entry
	curNode := f.root.findLongestPrefixEntryEnc(name)

	// Now step back up until we find a strategy entry
	// since some might only have a nexthops but no strategy
	var strategy enc.Name
	for ; curNode != nil; curNode = curNode.parent {
		if curNode.strategy != nil {
			strategy = curNode.strategy
			break
		}
	}

	return strategy
}

// InsertNextHop adds or updates a nexthop entry for the specified prefix.

func (f *FibStrategyTree) InsertNextHopEnc(name enc.Name, nexthop uint64, cost uint64) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()
	entry := f.fillTreeToPrefixEnc(name)
	if entry.name == nil {
		entry.name = name
	}
	for _, existingNexthop := range entry.nexthops {
		if existingNexthop.Nexthop == nexthop {
			existingNexthop.Cost = cost
			return
		}
	}

	newEntry := new(FibNextHopEntry)
	newEntry.Nexthop = nexthop
	newEntry.Cost = cost
	entry.nexthops = append(entry.nexthops, newEntry)
	var hash uint64
	hash = 0
	for _, component := range name {
		hash = hash + xxhash.Sum64(component.Val)
	}
	f.fibPrefixes[hash] = entry
}

// ClearNextHops clears all nexthops for the specified prefix.
func (f *FibStrategyTree) ClearNextHopsEnc(name enc.Name) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()

	if name == nil {
		return // In some wierd case, when RibEntry.updateNexthops() is called, the name becomes nil.
	}
	node := f.root.findExactMatchEntryEnc(name)
	if node != nil {
		node.nexthops = make([]*FibNextHopEntry, 0)
	}
}

// RemoveNextHop removes the specified nexthop entry from the specified prefix.

func (f *FibStrategyTree) RemoveNextHopEnc(name enc.Name, nexthop uint64) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()
	entry := f.root.findExactMatchEntryEnc(name)
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
			var hash uint64
			hash = 0
			for _, component := range name {
				hash = hash + xxhash.Sum64(component.Val)
			}
			delete(f.fibPrefixes, hash)
		}
		entry.pruneIfEmpty()
	}
}

// GetAllFIBEntries returns all nexthop entries in the FIB.
func (f *FibStrategyTree) GetAllFIBEntries() []FibStrategyEntry {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()

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
	return entries
}

// SetStrategy sets the strategy for the specified prefix.
func (f *FibStrategyTree) SetStrategyEnc(name enc.Name, strategy enc.Name) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()
	entry := f.fillTreeToPrefixEnc(name)
	if entry.name == nil {
		entry.name = name
	}
	entry.strategy = strategy
}

// UnsetStrategy unsets the strategy for the specified prefix.
func (f *FibStrategyTree) UnSetStrategyEnc(name enc.Name) {
	f.fibStrategyRWMutex.Lock()
	defer f.fibStrategyRWMutex.Unlock()
	entry := f.root.findExactMatchEntryEnc(name)
	if entry != nil {
		entry.strategy = nil
		entry.pruneIfEmptyEnc()
	}
}

// GetAllForwardingStrategies returns all strategy choice entries in the Strategy Table.
func (f *FibStrategyTree) GetAllForwardingStrategies() []FibStrategyEntry {
	f.fibStrategyRWMutex.RLock()
	defer f.fibStrategyRWMutex.RUnlock()

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
	return entries
}
