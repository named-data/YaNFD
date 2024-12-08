package dv

import (
	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

const Infinity = uint64(65535)

// Routing Information Base (RIB)
type rib struct {
	// destination hash -> name
	destinations map[uint64]enc.Name
	// destination hash -> entry
	entries map[uint64]*rib_entry
}

type rib_entry struct {
	// next hop face -> cost
	costs map[uint64]uint64
	// next hop for lowest cost
	nextHop1 uint64
	// lowest cost in this entry
	lowest1 uint64
	// second lowest cost in this entry
	lowest2 uint64
}

func newRib() *rib {
	return &rib{
		destinations: make(map[uint64]enc.Name),
		entries:      make(map[uint64]*rib_entry),
	}
}

// Set a destination in the RIB. Returns true if the Advertisement might change.
func (r *rib) set(destName enc.Name, nextHop uint64, cost uint64) bool {
	destHash := destName.Hash()
	r.destinations[destHash] = destName.Clone()

	entry, ok := r.entries[destHash]
	if !ok {
		entry = &rib_entry{
			costs: make(map[uint64]uint64),
		}
		r.entries[destHash] = entry
	}

	return entry.set(nextHop, cost)
}

// Get all advertisement entries in the RIB.
func (r *rib) advEntries() []*tlv.AdvEntry {
	entries := make([]*tlv.AdvEntry, 0, len(r.entries))

	for destHash, entry := range r.entries {
		destName := r.destinations[destHash]

		entries = append(entries, &tlv.AdvEntry{
			Destination: &tlv.Destination{
				Name: destName,
			},
			Interface: entry.nextHop1,
			Cost:      entry.lowest1,
			OtherCost: entry.lowest2,
		})
	}

	return entries
}

func (e *rib_entry) set(nextHop uint64, cost uint64) bool {
	if known, ok := e.costs[nextHop]; ok && known == cost {
		// Prevent triggering an unnecessary refresh
		return false
	}

	e.costs[nextHop] = cost
	return e.refresh()
}

// Update lowest and second lowest costs for the entry.
func (e *rib_entry) refresh() bool {
	lowest1 := Infinity
	lowest2 := Infinity
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
