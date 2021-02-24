/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/eric135/YaNFD/ndn/tlv"
)

// GeneralStatus contains status information about the forwarder's overall status.
type GeneralStatus struct {
	NfdVersion            string
	StartTimestamp        uint64
	CurrentTimestamp      uint64
	NNameTreeEntries      uint64
	NFibEntries           uint64
	NPitEntries           uint64
	NMeasurementEntries   uint64
	NCsEntries            uint64
	NInInterests          uint64
	NInData               uint64
	NInNacks              uint64
	NOutInterests         uint64
	NOutData              uint64
	NOutNacks             uint64
	NSatisfiedInterests   uint64
	NUnsatisfiedInterests uint64
}

// MakeGeneralStatus creates an empty GeneralStatus.
func MakeGeneralStatus() *GeneralStatus {
	g := new(GeneralStatus)
	return g
}

// Encode encodes a GeneralStatus.
func (g *GeneralStatus) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.Content) // We will need to extract the value to encode in a Data content field

	wire.Append(tlv.NewBlock(tlv.NfdVersion, []byte(g.NfdVersion)))
	wire.Append(tlv.EncodeNNIBlock(tlv.StartTimestamp, g.StartTimestamp))
	wire.Append(tlv.EncodeNNIBlock(tlv.CurrentTimestamp, g.CurrentTimestamp))
	wire.Append(tlv.EncodeNNIBlock(tlv.NNameTreeEntries, g.NNameTreeEntries))
	wire.Append(tlv.EncodeNNIBlock(tlv.NFibEntries, g.NFibEntries))
	wire.Append(tlv.EncodeNNIBlock(tlv.NPitEntries, g.NPitEntries))
	wire.Append(tlv.EncodeNNIBlock(tlv.NMeasurementEntries, g.NMeasurementEntries))
	wire.Append(tlv.EncodeNNIBlock(tlv.NCsEntries, g.NCsEntries))
	wire.Append(tlv.EncodeNNIBlock(tlv.NInInterests, g.NInInterests))
	wire.Append(tlv.EncodeNNIBlock(tlv.NInData, g.NInData))
	wire.Append(tlv.EncodeNNIBlock(tlv.NInNacks, g.NInNacks))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutInterests, g.NOutInterests))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutData, g.NOutData))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutNacks, g.NOutNacks))
	wire.Append(tlv.EncodeNNIBlock(tlv.NSatisfiedInterests, g.NSatisfiedInterests))
	wire.Append(tlv.EncodeNNIBlock(tlv.NUnsatisfiedInterests, g.NUnsatisfiedInterests))

	wire.Encode()
	return wire, nil
}
