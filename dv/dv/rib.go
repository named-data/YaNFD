package dv

import (
	"fmt"

	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// Routing Information Base (RIB)
type rib struct {
	// destination hash -> entry
	entries map[uint64]*rib_entry
	// neighbor hash -> neighbor name
	neighbors map[uint64]enc.Name
}

type rib_entry struct {
	// full name of destination router
	name enc.Name
	// neighbor hash -> cost
	costs map[uint64]uint64
	// next hop for lowest cost
	nextHop1 uint64
	// lowest cost in this entry
	lowest1 uint64
	// second lowest cost in this entry
	lowest2 uint64
	// needs refresh
	dirty bool
}

func NewRib() *rib {
	return &rib{
		entries:   make(map[uint64]*rib_entry),
		neighbors: make(map[uint64]enc.Name),
	}
}

// Print the RIB to the console (for debugging).
func (r *rib) Print() {
	for _, entry := range r.entries {
		fmt.Printf("=> Destination: %s\n", entry.name.String())
		for hop, cost := range entry.costs {
			if cost < CostInfinity {
				fmt.Printf("===> NextHop: %s, Cost: %d\n", r.neighbors[hop].String(), cost)
			}
		}
	}
}

// Set a destination in the RIB. Returns true if the Advertisement might change.
func (r *rib) set(destName enc.Name, nextHop enc.Name, cost uint64) bool {
	destHash := destName.Hash()
	nextHopHash := nextHop.Hash()

	// Create RIB entry if it doesn't exist
	entry, ok := r.entries[destHash]
	if !ok {
		entry = &rib_entry{
			name:  destName.Clone(),
			costs: make(map[uint64]uint64),
		}
		r.entries[destHash] = entry
	}

	// Create neighbor link if it doesn't exist
	if _, ok := r.neighbors[nextHopHash]; !ok {
		r.neighbors[nextHopHash] = nextHop.Clone()
	}

	return entry.set(nextHopHash, cost)
}

// Remove all entries with a given next hop.
// Returns true if the Advertisement might change.
func (r *rib) removeNextHop(nextHop enc.Name) bool {
	nextHopHash := nextHop.Hash()
	dirty := false

	for _, entry := range r.entries {
		if _, ok := entry.costs[nextHopHash]; ok {
			delete(entry.costs, nextHopHash)
			dirty = entry.refresh() || dirty
		}
	}

	return dirty
}

// Resets all entries for a given next hop to infinity without
// refreshing any entry. This is specifically intended for the
// RIB update algorithm to avoid unnecessary changes.
func (r *rib) dirtyResetNextHop(nextHop enc.Name) {
	nextHopHash := nextHop.Hash()
	for _, entry := range r.entries {
		entry.costs[nextHopHash] = CostInfinity
		entry.dirty = true
	}
}

// Whenever the RIB is changed, this must be called manually
// to remove unreachable destinations.
// Returns true if the Advertisement might change.
func (r *rib) prune() bool {
	dirty := false
	for _, entry := range r.entries {
		// Refresh entry if dirty
		if entry.dirty {
			dirty = entry.refresh() || dirty
		}

		// Remove if no valid next hops
		if entry.lowest1 == CostInfinity {
			delete(r.entries, entry.name.Hash())
			dirty = true
		}
	}
	return dirty
}

// Get all advertisement entries in the RIB.
func (r *rib) advert() *tlv.Advertisement {
	advert := &tlv.Advertisement{
		Entries: make([]*tlv.AdvEntry, 0, len(r.entries)),
	}

	for _, entry := range r.entries {
		advert.Entries = append(advert.Entries, &tlv.AdvEntry{
			Destination: &tlv.Destination{Name: entry.name},
			NextHop: &tlv.Destination{
				Name: r.neighbors[entry.nextHop1],
			},
			Cost:      entry.lowest1,
			OtherCost: entry.lowest2,
		})
	}

	return advert
}

func (e *rib_entry) set(nextHop uint64, cost uint64) bool {
	if known, ok := e.costs[nextHop]; !ok || known != cost {
		e.costs[nextHop] = cost
		return e.refresh()
	}

	// Prevent triggering an unnecessary refresh
	return false
}

// Update lowest and second lowest costs for the entry.
func (e *rib_entry) refresh() bool {
	e.dirty = false
	lowest1 := CostInfinity
	lowest2 := CostInfinity
	nextHop1 := uint64(0)

	for hop, cost := range e.costs {
		if cost < lowest1 {
			lowest2 = lowest1
			lowest1 = cost
			nextHop1 = hop
		} else if cost < lowest2 {
			lowest2 = cost
		}
	}

	if e.lowest1 != lowest1 || e.lowest2 != lowest2 || e.nextHop1 != nextHop1 {
		e.lowest1 = lowest1
		e.lowest2 = lowest2
		e.nextHop1 = nextHop1
		return true
	}

	return false
}
