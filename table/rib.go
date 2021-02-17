/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"container/list"
	"time"

	"github.com/eric135/YaNFD/ndn"
)

// RibEntry represents an entry in the RIB table.
type RibEntry struct {
	component ndn.NameComponent
	Name      *ndn.Name
	depth     int

	parent   *RibEntry
	children []*RibEntry

	routes []*Route
}

// Route represents a route in a RIB entry.
type Route struct {
	FaceID           uint64
	Origin           uint64
	Cost             uint64
	Flags            uint64
	ExpirationPeriod *time.Duration
}

// Route flags.
const (
	RouteFlagChildInherit uint64 = 0x01
	RouteFlagCapture      uint64 = 0x02
)

// Route origins.
const (
	RouteOriginApp       uint64 = 0
	RouteOriginStatic    uint64 = 255
	RouteOriginNLSR      uint64 = 128
	RouteOriginPrefixAnn uint64 = 129
	RouteOriginClient    uint64 = 65
	RouteOriginAutoreg   uint64 = 64
	RouteOriginAutoconf  uint64 = 66
)

// Rib is a table containing the RIB.
var Rib *RibEntry
var ribPrefixes map[string]*RibEntry

func init() {
	Rib = new(RibEntry)
	Rib.component = nil // Root component will be nil since it represents zero components
	ribPrefixes = make(map[string]*RibEntry)
}

func (r *RibEntry) findExactMatchEntry(name *ndn.Name) *RibEntry {
	if name.Size() > r.depth {
		for _, child := range r.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findExactMatchEntry(name)
			}
		}
	} else if name.Size() == r.depth {
		return r
	}
	return nil
}

func (r *RibEntry) findLongestPrefixEntry(name *ndn.Name) *RibEntry {
	if name.Size() > r.depth {
		for _, child := range r.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findLongestPrefixEntry(name)
			}
		}
	}
	return r
}

func (r *RibEntry) fillTreeToPrefix(name *ndn.Name) *RibEntry {
	curNode := r.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(RibEntry)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		curNode.children = append(curNode.children, newNode)
		curNode = newNode
	}
	return curNode
}

func (r *RibEntry) pruneIfEmpty() {
	for curNode := r; curNode.parent != nil && len(curNode.children) == 0 && len(curNode.routes) == 0; curNode = curNode.parent {
		// Remove from parent's children
		for i, child := range curNode.parent.children {
			if child == r {
				if i < len(curNode.parent.children)-1 {
					copy(curNode.parent.children[i:], curNode.parent.children[i+1:])
				}
				curNode.parent.children = curNode.parent.children[:len(curNode.parent.children)-1]
				break
			}
		}
	}
}

func (r *RibEntry) updateNexthops(node *RibEntry) {
	FibStrategyTable.ClearNexthops(node.Name)

	// Find minimum cost route per nexthop
	minCostRoutes := make(map[uint64]uint64) // FaceID -> Cost
	for _, route := range node.routes {
		if _, ok := minCostRoutes[route.FaceID]; !ok {
			minCostRoutes[route.FaceID] = route.Cost
		} else if route.Cost < minCostRoutes[route.FaceID] {
			minCostRoutes[route.FaceID] = route.Cost
		}
	}

	// Add "flattened" set of nexthops
	for nexthop, cost := range minCostRoutes {
		FibStrategyTable.AddNexthop(node.Name, nexthop, cost)
	}
}

// AddRoute adds or updates a RIB entry for the specified prefix.
func (r *RibEntry) AddRoute(name *ndn.Name, faceID uint64, origin uint64, cost uint64, flags uint64, expirationPeriod *time.Duration) {
	node := r.fillTreeToPrefix(name)
	if node.Name == nil {
		node.Name = name
	}
	var route *Route
	for _, existingRoute := range node.routes {
		if existingRoute.FaceID == faceID && existingRoute.Origin == origin {
			existingRoute.Cost = cost
			existingRoute.Flags = flags
			existingRoute.ExpirationPeriod = expirationPeriod
			route = existingRoute
			break
		}
	}

	if route == nil {
		newEntry := new(Route)
		newEntry.FaceID = faceID
		newEntry.Origin = origin
		newEntry.Cost = cost
		newEntry.Flags = flags
		newEntry.ExpirationPeriod = expirationPeriod
		node.routes = append(node.routes, newEntry)
		ribPrefixes[name.String()] = node
	}

	r.updateNexthops(node)
}

// GetAllEntries returns all routes in the FIB.
func (r *RibEntry) GetAllEntries() []*RibEntry {
	entries := make([]*RibEntry, 0)
	// Walk tree in-order
	queue := list.New()
	queue.PushBack(r)
	for queue.Len() > 0 {
		ribEntry := queue.Front().Value.(*RibEntry)
		queue.Remove(queue.Front())
		// Add all children to stack
		for _, child := range ribEntry.children {
			queue.PushFront(child)
		}

		// If has any routes, add to list
		if len(ribEntry.routes) > 0 {
			entries = append(entries, ribEntry)
		}
	}
	return entries
}

// GetRoutes returns all routes in the RIB entry.
func (r *RibEntry) GetRoutes() []*Route {
	return r.routes
}

// RemoveRoute removes the specified route from the specified prefix.
func (r *RibEntry) RemoveRoute(name *ndn.Name, faceID uint64, origin uint64) {
	entry := r.findExactMatchEntry(name)
	if entry != nil {
		for i, existingRoute := range entry.routes {
			if existingRoute.FaceID == faceID && existingRoute.Origin == origin {
				if i < len(entry.routes)-1 {
					copy(entry.routes[i:], entry.routes[i+1:])
				}
				entry.routes = entry.routes[:len(entry.routes)-1]
				break
			}
		}
		if len(entry.routes) == 0 {
			delete(ribPrefixes, name.String())
		}
		r.updateNexthops(entry)
		entry.pruneIfEmpty()
	}
}
