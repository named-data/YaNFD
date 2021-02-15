/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// FibEntry contains status information about a FIB entry.
type FibEntry struct {
	Name     ndn.Name
	Nexthops []NextHopRecord
}

// NextHopRecord represents a next hop record in a FibEntry.
type NextHopRecord struct {
	FaceID uint64
	Cost   uint64
}

// MakeFibEntry creates an empty FibEntry.
func MakeFibEntry(name *ndn.Name) *FibEntry {
	f := new(FibEntry)
	f.Name = *name
	f.Nexthops = make([]NextHopRecord, 0)
	return f
}

// Encode encodes a FibEntry.
func (f *FibEntry) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.FibEntry)

	wire.Append(f.Name.Encode())

	for _, record := range f.Nexthops {
		nexthopWire := tlv.NewEmptyBlock(tlv.NextHopRecord)
		nexthopWire.Append(tlv.EncodeNNIBlock(tlv.FaceID, record.FaceID))
		nexthopWire.Append(tlv.EncodeNNIBlock(tlv.Cost, record.Cost))
		wire.Append(nexthopWire)
	}

	wire.Encode()
	return wire, nil
}
