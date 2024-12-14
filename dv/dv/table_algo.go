package dv

import (
	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/table"
	"github.com/zjkmxy/go-ndn/pkg/log"
)

// Compute the RIB chnages for this neighbor
func (dv *Router) ribUpdate(ns *table.NeighborState) {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	if ns.Advert == nil {
		return
	}

	// TODO: use cost to neighbor
	localCost := uint64(1)

	// Trigger our own advertisement if needed
	var dirty bool = false

	// Reset destinations for this neighbor
	dv.rib.DirtyResetNextHop(ns.Name)

	for _, entry := range ns.Advert.Entries {
		// Use the advertised cost by default
		cost := entry.Cost + localCost

		// Poison reverse - try other cost if next hop is us
		if entry.NextHop.Name.Equal(dv.config.RouterPfxN) {
			if entry.OtherCost < config.CostInfinity {
				cost = entry.OtherCost + localCost
			} else {
				cost = config.CostInfinity
			}
		}

		// Skip unreachable destinations
		if cost >= config.CostInfinity {
			continue
		}

		// Check advertisement changes
		dirty = dv.rib.Set(entry.Destination.Name, ns.Name, cost) || dirty
	}

	// Drop dead entries
	dirty = dv.rib.Prune() || dirty

	// If advert changed, increment sequence number
	if dirty {
		go dv.advertSyncNotifyNew()
		go dv.prefixDataFetchAll()
	}
}

// Check for dead neighbors
func (dv *Router) checkDeadNeighbors() {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	dirty := false
	for _, ns := range dv.neighbors.GetAll() {
		// Check if the neighbor is entirely dead
		if ns.IsDead() {
			log.Infof("checkDeadNeighbors: Neighbor %s is dead", ns.Name.String())

			// This is the ONLY place that can remove neighbors
			dv.neighbors.Remove(ns.Name)

			// Remove neighbor from RIB and prune
			dirty = dv.rib.RemoveNextHop(ns.Name) || dirty
			dirty = dv.rib.Prune() || dirty
		}
	}

	if dirty {
		go dv.advertSyncNotifyNew()
	}
}
