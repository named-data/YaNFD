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

// producerRegions contains the prefixes produced in this forwarder's region.
var producerRegions []string

// Configure configures the forwarding system.
func Configure() {
	tableQueueSize = core.GetConfigIntDefault("tables.queue_size", 1024)
	deadNonceListLifetime = time.Duration(core.GetConfigIntDefault("tables.dead_nonce_list.lifetime", 6000)) * time.Millisecond
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
