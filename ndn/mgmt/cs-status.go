/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/named-data/YaNFD/ndn/tlv"
)

// CsFlag indicates a ContentStore status flag.
type CsFlag int

const (
	CsFlagEnableAdmit CsFlag = 1 << iota
	CsFlagEnableServe
)

// CsStatus contains status information about the Content Store.
type CsStatus struct {
	Capacity   uint64
	Flags      CsFlag
	NCsEntries uint64
	NHits      uint64
	NMisses    uint64
}

// Encode encodes a CsStatus.
func (s *CsStatus) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.CsInfo)
	wire.Append(tlv.EncodeNNIBlock(tlv.Capacity, s.Capacity))
	wire.Append(tlv.EncodeNNIBlock(tlv.Flags, uint64(s.Flags)))
	wire.Append(tlv.EncodeNNIBlock(tlv.NCsEntries, s.NCsEntries))
	wire.Append(tlv.EncodeNNIBlock(tlv.NHits, s.NHits))
	wire.Append(tlv.EncodeNNIBlock(tlv.NMisses, s.NMisses))
	return wire, wire.Encode()
}
