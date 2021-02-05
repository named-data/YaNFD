/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"net"
	"strconv"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// MulticastEthernetTransport is a multicast Ethernet transport.
type MulticastEthernetTransport struct {
	pcap       *pcap.Handle
	shouldQuit chan bool
	remoteAddr net.HardwareAddr
	localAddr  net.HardwareAddr
	transportBase
}

// MakeMulticastEthernetTransport creates a new multicast Ethernet transport.
func MakeMulticastEthernetTransport(remoteURI *ndn.URI, localURI *ndn.URI) (*MulticastEthernetTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "ether" || !localURI.IsCanonical() || localURI.Scheme() != "dev" {
		return nil, core.ErrNotCanonical
	}

	var t MulticastEthernetTransport
	t.makeTransportBase(remoteURI, localURI, tlv.MaxNDNPacketSize)
	t.shouldQuit = make(chan bool, 1)
	var err error
	t.remoteAddr, err = net.ParseMAC(remoteURI.Path())
	if err != nil {
		core.LogError(t, "Unable to parse MAC address", remoteURI.Path(), "-", err)
		return nil, err
	}

	// Get interface
	iface, err := net.InterfaceByName(localURI.Path())
	if err != nil {
		core.LogError(t, "Unable to get local interface", localURI.Path(), "-", err)
		return nil, err
	}
	t.localAddr = iface.HardwareAddr

	// Set scope
	t.scope = ndn.NonLocal

	// Set up inactive PCAP handle
	inactive, err := pcap.NewInactiveHandle(localURI.Path())
	if err != nil {
		core.LogError(t, "Unable to create PCAP handle", err)
		return nil, err
	}

	err = inactive.SetTimeout(time.Minute)
	if err != nil {
		core.LogError(t, "Unable to set PCAP timeout", err)
		return nil, err
	}

	// Activate PCAP handle
	t.pcap, err = inactive.Activate()

	// Set PCAP filter
	err = t.pcap.SetBPFFilter("ether proto " + strconv.Itoa(NDNEtherType) + " and ether dst " + remoteURI.Path())
	if err != nil {
		core.LogError(t, "Unable to set PCAP filter", err)
	}

	t.changeState(ndn.Up)

	return &t, nil
}

func (t *MulticastEthernetTransport) String() string {
	return "MulticastEthernetTransport, FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

func (t *MulticastEthernetTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	// Wrap in Ethernet frame
	ethHeader := layers.Ethernet{SrcMAC: t.localAddr, DstMAC: t.remoteAddr, EthernetType: NDNEtherType}
	ethFrame := gopacket.NewSerializeBuffer()
	gopacket.SerializeLayers(ethFrame, gopacket.SerializeOptions{}, &ethHeader, gopacket.Payload(frame))

	// Write to PCAP handle
	core.LogDebug(t, "Sending frame of size", len(ethFrame.Bytes()))
	err := t.pcap.WritePacketData(ethFrame.Bytes())
	if err != nil {
		core.LogWarn(t, "Unable to write frame - DROP and FACE DOWN")
		t.changeState(ndn.Down)
	}
}

func (t *MulticastEthernetTransport) runReceive() {
	packetSource := gopacket.NewPacketSource(t.pcap, t.pcap.LinkType())
	for {
		select {
		case packet := <-packetSource.Packets():
			core.LogDebug(t, "Received", len(packet.Data()), "bytes from", packet.LinkLayer().LinkFlow().Src().String())

			// Extract network layer (NDN)
			ndnLayer := packet.NetworkLayer().LayerContents()

			if len(ndnLayer) > tlv.MaxNDNPacketSize {
				core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			}

			// Determine whether valid packet received
			_, _, tlvSize, err := tlv.DecodeTypeLength(ndnLayer)
			if err != nil {
				core.LogInfo("Unable to process received frame: " + err.Error() + " - DROP")
			} else if len(ndnLayer) >= tlvSize {
				// Packet was successfully received, send up to link service
				t.linkService.handleIncomingFrame(ndnLayer[:tlvSize])
			} else {
				core.LogInfo("Received frame is incomplete - DROP")
			}
		case <-t.shouldQuit:
			return
		}
	}
}

func (t *MulticastEthernetTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "- state:", t.state, "->", new)
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing unicast Ethernet transport")
		t.shouldQuit <- true

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
