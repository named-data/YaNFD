//go:build windows || cgo

/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn/tlv"
)

// OpenPcap creates and activates a PCAP handle.
func OpenPcap(device, bpfFilter string) (PcapHandle, error) {
	// Set up inactive PCAP handle
	inactive, err := pcap.NewInactiveHandle(device)
	if err != nil {
		core.LogError("Face-Pcap", "Unable to create PCAP handle: ", err)
		return nil, err
	}
	defer inactive.CleanUp()

	// Set snap length (max amount of frame to capture)
	if err := inactive.SetSnapLen(18 + tlv.MaxNDNPacketSize); err != nil {
		core.LogError("Face-Pcap", "Unable to set PCAP snap length: ", err)
		return nil, err
	}

	// Enable immediate mode
	if err := inactive.SetImmediateMode(true); err != nil {
		core.LogError("Face-Pcap", "Unable to set immediate mode for PCAP capture: ", err)
		return nil, err
	}

	// Set PCAP buffer size to 24 MB
	if err := inactive.SetBufferSize(24 * 1024 * 1024); err != nil {
		core.LogError("Face-Pcap", "Unable to set buffer size for PCAP capture: ", err)
		return nil, err
	}

	// Activate PCAP handle
	hdl, err := inactive.Activate()
	if err != nil {
		core.LogError("Face-Pcap", "Unable to activate PCAP handle: ", err)
		return nil, err
	}

	// Set PCAP direction to in
	if err := hdl.SetDirection(pcap.DirectionIn); err != nil {
		core.LogError("Face-Pcap", "Unable to set direction for PCAP handle: ", err)
		return nil, err
	}

	// Set PCAP data link type to EN10MB
	if err := hdl.SetLinkType(layers.LinkTypeEthernet); err != nil {
		core.LogError("Face-Pcap", "Unable to set data link type for PCAP handle: ", err)
		return nil, err
	}

	// Set PCAP filter (string based on NFD's formatting string)
	if err := hdl.SetBPFFilter(bpfFilter); err != nil {
		core.LogError("Face-Pcap", "Unable to set PCAP filter: ", err)
	}

	return hdl, nil
}
