/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"container/list"
	"time"

	"github.com/named-data/YaNFD/ndn"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// RibTable represents the Routing Information Base (RIB).
type RibTable struct {
	RibEntry
}

// RibEntry represents an entry in the RIB table.
type RibEntry struct {
	component    ndn.NameComponent
	encComponent enc.Component
	Name         *ndn.Name
	EncName      *enc.Name
	depth        int

	parent   *RibEntry
	children map[*RibEntry]bool

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

// Rib is the Routing Information Base.
var Rib = RibTable{
	RibEntry: RibEntry{
		children: map[*RibEntry]bool{},
	},
}

func (r *RibTable) fillTreeToPrefixEnc(name *enc.Name) *RibEntry {
	entry := r.findLongestPrefixEntryEnc(name)
	for depth := entry.depth + 1; depth <= len(*name); depth++ {
		child := &RibEntry{
			depth:        depth,
			encComponent: deepCopy(At(name, depth-1)),
			parent:       entry,
			children:     map[*RibEntry]bool{},
		}
		entry.children[child] = true
		entry = child
	}
	return entry
}
func (r *RibEntry) findExactMatchEntryEnc(name *enc.Name) *RibEntry {
	if len(*name) > r.depth {
		for child := range r.children {
			if At(name, child.depth-1).Equal(child.encComponent) {
				return child.findExactMatchEntryEnc(name)
			}
		}
	} else if len(*name) == r.depth {
		return r
	}
	return nil
}

func (r *RibEntry) findLongestPrefixEntryEnc(name *enc.Name) *RibEntry {
	if len(*name) > r.depth {
		for child := range r.children {
			if At(name, child.depth-1).Equal(child.encComponent) {
				return child.findLongestPrefixEntryEnc(name)
			}
		}
	}
	return r
}

func (r *RibEntry) pruneIfEmpty() {
	for entry := r; entry.parent != nil && len(entry.children) == 0 && len(entry.routes) == 0; entry = entry.parent {
		// Remove from parent's children
		delete(entry.parent.children, entry)
	}
}
func (r *RibEntry) updateNexthopsEnc() {
	FibStrategyTable.ClearNextHopsEnc(r.EncName)

	// Find minimum cost route per nexthop
	minCostRoutes := make(map[uint64]uint64) // FaceID -> Cost
	for _, route := range r.routes {
		cost, ok := minCostRoutes[route.FaceID]
		if !ok || route.Cost < cost {
			minCostRoutes[route.FaceID] = route.Cost
		}
	}

	// Add "flattened" set of nexthops
	for nexthop, cost := range minCostRoutes {
		FibStrategyTable.InsertNextHopEnc(r.EncName, nexthop, cost)
	}
}

// AddRoute adds or updates a RIB entry for the specified prefix.
func (r *RibTable) AddEncRoute(name *enc.Name, faceID uint64, origin uint64, cost uint64, flags uint64, expirationPeriod *time.Duration) {
	node := r.fillTreeToPrefixEnc(name)
	if node.EncName == nil {
		node.EncName = name
	}

	defer node.updateNexthopsEnc()

	for _, existingRoute := range node.routes {
		if existingRoute.FaceID == faceID && existingRoute.Origin == origin {
			existingRoute.Cost = cost
			existingRoute.Flags = flags
			existingRoute.ExpirationPeriod = expirationPeriod
			return
		}
	}

	node.routes = append(node.routes, &Route{
		FaceID:           faceID,
		Origin:           origin,
		Cost:             cost,
		Flags:            flags,
		ExpirationPeriod: expirationPeriod,
	})
}

// GetAllEntries returns all routes in the RIB.
func (r *RibTable) GetAllEntries() []*RibEntry {
	entries := make([]*RibEntry, 0)
	// Walk tree in-order
	queue := list.New()
	queue.PushBack(&r.RibEntry)
	for queue.Len() > 0 {
		ribEntry := queue.Front().Value.(*RibEntry)
		queue.Remove(queue.Front())
		// Add all children to stack
		for child := range ribEntry.children {
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
func (r *RibTable) RemoveRouteEnc(name *enc.Name, faceID uint64, origin uint64) {
	entry := r.findExactMatchEntryEnc(name)
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
		entry.updateNexthopsEnc()
		entry.pruneIfEmpty()
	}
}

// CleanUpFace removes the specified face from all entries. Used for clean-up after a face is destroyed.
func (r *RibEntry) CleanUpFace(faceId uint64) {
	// Recursively clean children
	for child := range r.children {
		child.CleanUpFace(faceId)
	}

	if r.EncName == nil {
		return
	}
	for i, existingNexthop := range r.routes {
		if existingNexthop.FaceID == faceId {
			if i < len(r.routes)-1 {
				copy(r.routes[i:], r.routes[i+1:])
			}
			r.routes = r.routes[:len(r.routes)-1]
			break
		}
	}
	r.updateNexthopsEnc()
	r.pruneIfEmpty()
}
