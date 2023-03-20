/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"time"

	"github.com/named-data/YaNFD/core"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// tableQueueSize is the maxmimum size of queues in the tables.
var tableQueueSize int

// deadNonceListLifetime is the lifetime of entries in the dead nonce list.
var deadNonceListLifetime time.Duration

// csCapacity contains the default capacity of each forwarding thread's Content Store.
var csCapacity int

// csAdmit determines whether contents will be admitted to the Content Store.
var csAdmit bool

// csServe determines whether contents will be served from the Content Store.
var csServe bool

// csReplacementPolicy contains the replacement policy used by Content Stores in the forwarder.
var csReplacementPolicy string

// producerRegions contains the prefixes produced in this forwarder's region.
var producerRegions []string

// fibTableAlgorithm contains the options for how the FIB is implemented
// Allowed values: nametree, hashtable
var fibTableAlgorithm string

// Configure configures the forwarding system.
func Configure() {
	tableQueueSize = core.GetConfigIntDefault("tables.queue_size", 1024)

	// Content Store
	csCapacity = int(core.GetConfigUint16Default("tables.content_store.capacity", 1024))
	csAdmit = core.GetConfigBoolDefault("tables.content_store.admit", true)
	csServe = core.GetConfigBoolDefault("tables.content_store.serve", true)
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
		name, err := enc.NameFromStr(region)
		if err != nil {
			core.LogFatal("NetworkRegionTable", "Could not add name=", region, " to table: ", err)
		}
		NetworkRegion.Add(&name)
		core.LogDebug("NetworkRegionTable", "Added name=", region, " to table")
	}
}

// SetCsCapacity sets the CS capacity from management.
func SetCsCapacity(capacity int) {
	csCapacity = capacity
}

func CreateFIBTable(fibTableAlgorithm string) {
	switch fibTableAlgorithm {
	case "nametree":
		newFibStrategyTableTree()
	default:
		// Default to nametree
		core.LogFatal("CreateFIBTable", "Unrecognized FIB table algorithm specified: ", fibTableAlgorithm)
	}
}
