/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import "github.com/named-data/YaNFD/ndn/tlv"

// PendingPacket represents a pending network-layer packet to be sent or recently received on the link, plus any associated metadata.
type PendingPacket struct {
	Wire           *tlv.Block
	NetPacket      interface{}
	PitToken       []byte
	CongestionMark *uint64
	IncomingFaceID *uint64
	NextHopFaceID  *uint64
	CachePolicy    *uint64
}

// DeepCopy creates a deep copy of a pending packet.
func (p *PendingPacket) DeepCopy() *PendingPacket {
	newP := new(PendingPacket)
	// Deep copy not needed because wire it not used in incoming Data pipeline (only place where DeepCopy called)
	newP.Wire = p.Wire
	// Don't need to deep copy because only deep copied for Data packets from local producers and Data packets won't change in different threads
	newP.NetPacket = p.NetPacket
	newP.PitToken = p.PitToken
	newP.CongestionMark = p.CongestionMark
	newP.IncomingFaceID = p.IncomingFaceID
	newP.NextHopFaceID = p.NextHopFaceID
	newP.CachePolicy = p.CachePolicy
	return newP
}
