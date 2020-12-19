/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package lpv2

// TLV types for NDNLPv2.
const (
	Fragment           = 0x50
	Sequence           = 0x51
	FragIndex          = 0x52
	FragCount          = 0x53
	PitToken           = 0x62
	LpPacket           = 0x64
	Nack               = 0x0320
	NextHopFaceID      = 0x0330
	IncomingFaceID     = 0x0331
	CachePolicy        = 0x0334
	CachePolicyType    = 0x0335
	CongestionMark     = 0x0340
	Ack                = 0x0344
	TxSequence         = 0x0348
	NonDiscovery       = 0x034c
	PrefixAnnouncement = 0x0350
)

// IsCritical returns whether the NDNLPv2 TLV type is critical.
func IsCritical(tlvType uint32) bool {
	if tlvType >= 800 && tlvType <= 959 {
		return tlvType&0x3 == 0
	}
	return true
}
