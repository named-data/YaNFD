/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// PcapHandle contains a subset of *pcap.Handle methods.
type PcapHandle interface {
	gopacket.PacketDataSource
	LinkType() layers.LinkType
	WritePacketData(data []byte) error
	Close()
}
