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
	tableQueueSize = core.GetConfig().Tables.QueueSize

	// Content Store
	csCapacity = int(core.GetConfig().Tables.ContentStore.Capacity)
	csAdmit = core.GetConfig().Tables.ContentStore.Admit
	csServe = core.GetConfig().Tables.ContentStore.Serve
	csReplacementPolicyName := core.GetConfig().Tables.ContentStore.ReplacementPolicy
	switch csReplacementPolicyName {
	case "lru":
		csReplacementPolicy = "lru"
	default:
		// Default to LRU
		csReplacementPolicy = "lru"
	}

	// Dead Nonce List
	deadNonceListLifetime = time.Duration(core.GetConfig().Tables.DeadNonceList.Lifetime) * time.Millisecond

	// Network Region Table
	producerRegions = core.GetConfig().Tables.NetworkRegion.Regions
	if producerRegions == nil {
		producerRegions = make([]string, 0)
	}
	for _, region := range producerRegions {
		name, err := enc.NameFromStr(region)
		if err != nil {
			core.LogFatal("NetworkRegionTable", "Could not add name=", region, " to table: ", err)
		}
		NetworkRegion.Add(name)
		core.LogDebug("NetworkRegionTable", "Added name=", region, " to table")
	}
}

// SetCsCapacity sets the CS capacity from management.
func SetCsCapacity(capacity int) {
	csCapacity = capacity
}

// CsCapacity returns the CS capacity
func CsCapacity() int {
	return csCapacity
}

func CreateFIBTable(fibTableAlgorithm string) {
	switch fibTableAlgorithm {
	case "hashtable":
		m := core.GetConfig().Tables.Fib.Hashtable.M
		newFibStrategyTableHashTable(m)
	case "nametree":
		newFibStrategyTableTree()
	default:
		// Default to nametree
		core.LogFatal("CreateFIBTable", "Unrecognized FIB table algorithm specified: ", fibTableAlgorithm)
	}
}
