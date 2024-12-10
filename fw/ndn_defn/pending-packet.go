/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_defn

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
)

// PendingPacket represents a pending network-layer packet to be sent
// or recently received on the link, plus any associated metadata.
type PendingPacket struct {
	Name      enc.Name
	EncPacket *spec.Packet
	RawBytes  []byte

	PitToken       []byte
	CongestionMark *uint64
	IncomingFaceID *uint64
	NextHopFaceID  *uint64
	CachePolicy    *uint64
}
