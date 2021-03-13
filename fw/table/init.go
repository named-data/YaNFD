/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
)

// tableQueueSize is the maxmimum size of queues in the tables.
var tableQueueSize int

// deadNonceListLifetime is the lifetime of entries in the dead nonce list.
var deadNonceListLifetime time.Duration

// csCapacity contains the default capacity of each forwarding thread's Content Store.
var csCapacity int

// csReplacementPolicy contains the replacement policy used by Content Stores in the forwarder.
var csReplacementPolicy string

// producerRegions contains the prefixes produced in this forwarder's region.
var producerRegions []string

// Configure configures the forwarding system.
func Configure() {
	tableQueueSize = core.GetConfigIntDefault("tables.queue_size", 1024)

	// Content Store
	csCapacity = int(core.GetConfigUint16Default("tables.content_store.capacity", 1024))
	csReplacementPolicyName := core.GetConfigStringDefault("tables.content_store.replacement_policy", "lru")
	switch csReplacementPolicyName {
	case "lru":
		csReplacementPolicy = "lru"
	default:
		// Default to LRU
		csReplacementPolicy = "lru"
	}

	// Dead Nonce List
	deadNonceListLifetime = time.Duration(core.GetConfigIntDefault("tables.dead_nonce_list.lifetime", 6000)) * time.Millisecond

	// Network Region Table
	producerRegions = core.GetConfigArrayString("tables.network_region.regions")
	if producerRegions == nil {
		producerRegions = make([]string, 0)
	}
	for _, region := range producerRegions {
		name, err := ndn.NameFromString(region)
		if err != nil {
			core.LogFatal("NetworkRegionTable", "Could not add name="+region+" to table: "+err.Error())
		}
		NetworkRegion.Add(name)
		core.LogDebug("NetworkRegionTable", "Added name="+region+" to table")
	}
}

// SetCsCapacity sets the CS capacity from management.
func SetCsCapacity(capacity int) {
	csCapacity = capacity
}
