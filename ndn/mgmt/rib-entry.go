/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"time"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/tlv"
)

// RibEntry contains status information about a RIB entry.
type RibEntry struct {
	Name   ndn.Name
	Routes []Route
}

// Route represents a route record in a RibEntry.
type Route struct {
	FaceID           uint64
	Origin           uint64
	Cost             uint64
	Flags            uint64
	ExpirationPeriod *time.Duration
}

// MakeRibEntry creates an empty RibEntry.
func MakeRibEntry(name *ndn.Name) *RibEntry {
	f := new(RibEntry)
	f.Name = *name
	f.Routes = make([]Route, 0)
	return f
}

// Encode encodes a RibEntry.
func (f *RibEntry) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.RibEntry)

	wire.Append(f.Name.Encode())

	for _, record := range f.Routes {
		nexthopWire := tlv.NewEmptyBlock(tlv.Route)
		nexthopWire.Append(tlv.EncodeNNIBlock(tlv.FaceID, record.FaceID))
		nexthopWire.Append(tlv.EncodeNNIBlock(tlv.Origin, record.Origin))
		nexthopWire.Append(tlv.EncodeNNIBlock(tlv.Cost, record.Cost))
		nexthopWire.Append(tlv.EncodeNNIBlock(tlv.Flags, record.Flags))
		if record.ExpirationPeriod != nil {
			nexthopWire.Append(tlv.EncodeNNIBlock(tlv.FaceID, uint64(record.ExpirationPeriod.Milliseconds())))
		}
		wire.Append(nexthopWire)
	}

	wire.Encode()
	return wire, nil
}
