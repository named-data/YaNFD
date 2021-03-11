/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import "github.com/eric135/YaNFD/ndn/tlv"

// PendingPacket represents a pending network-layer packet to be sent or recently received on the link, plus any associated metadata.
type PendingPacket struct {
	Wire           *tlv.Block
	PitToken       []byte
	CongestionMark *uint64
	IncomingFaceID *uint64
	NextHopFaceID  *uint64
	CachePolicy    *uint64
}

// DeepCopy creates a deep copy of a pending packet.
func (p *PendingPacket) DeepCopy() *PendingPacket {
	newP := new(PendingPacket)
	if p.Wire != nil {
		newP.Wire = p.Wire.DeepCopy()
	}
	newP.PitToken = make([]byte, len(p.PitToken))
	copy(newP.PitToken, p.PitToken)
	if p.CongestionMark != nil {
		newP.CongestionMark = new(uint64)
		*newP.CongestionMark = *p.CongestionMark
	}
	if p.IncomingFaceID != nil {
		newP.IncomingFaceID = new(uint64)
		*newP.IncomingFaceID = *p.IncomingFaceID
	}
	if p.NextHopFaceID != nil {
		newP.NextHopFaceID = new(uint64)
		*newP.NextHopFaceID = *p.NextHopFaceID
	}
	if p.CachePolicy != nil {
		newP.CachePolicy = new(uint64)
		*newP.CachePolicy = *p.CachePolicy
	}
	return newP
}
