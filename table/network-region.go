/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"github.com/named-data/YaNFD/ndn"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type networkRegionTable struct {
	table    []*enc.Name
	oldtable []*ndn.Name
}

// NetworkRegion contains producer region names for this forwarder..
var NetworkRegion *networkRegionTable

func init() {
	NetworkRegion = new(networkRegionTable)
}

// Add adds a name to the network region table.
func (n *networkRegionTable) Add(name *enc.Name) {
	for _, region := range n.table {
		if region.Equal(*name) {
			return
		}
	}
	n.table = append(n.table, name)
}

// IsProducer returns whether an entry in the network region table is a prefix of the specified name.
func (n *networkRegionTable) IsProducer(name *enc.Name) bool {
	for _, region := range n.table {
		if region.IsPrefix(*name) {
			return true
		}
	}
	return false
}

func (n *networkRegionTable) OldAdd(name *ndn.Name) {
	for _, region := range n.oldtable {
		if region.Equals(name) {
			return
		}
	}
	n.oldtable = append(n.oldtable, name)
}

// IsProducer returns whether an entry in the network region table is a prefix of the specified name.
func (n *networkRegionTable) OldIsProducer(name *ndn.Name) bool {
	for _, region := range n.oldtable {
		if region.PrefixOf(name) {
			return true
		}
	}
	return false
}
