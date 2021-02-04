/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"errors"
	"net"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face/impl"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// MulticastUDPTransport is a multicast UDP transport.
type MulticastUDPTransport struct {
	sendConn  net.Conn
	recvConn  *net.UDPConn
	groupAddr net.UDPAddr
	localAddr net.UDPAddr
	isIPv4    bool
	transportBase
}

// MakeMulticastUDPTransport creates a new multicast UDP transport.
func MakeMulticastUDPTransport(localURI ndn.URI) (*MulticastUDPTransport, error) {
	// Validate local URI
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	t := new(MulticastUDPTransport)
	// Get local interface
	localIf, err := InterfaceByIP(net.ParseIP(localURI.PathHost()))
	if err != nil || localIf == nil {
		core.LogError(t, "Unable to get interface for local URI", localURI.String(), ":", err)
	}

	if localURI.Scheme() == "udp4" {
		t.makeTransportBase(ndn.DecodeURIString(NDNMulticastUDP4URI), localURI, tlv.MaxNDNPacketSize)
	} else if localURI.Scheme() == "udp6" {
		t.makeTransportBase(ndn.DecodeURIString("udp6://["+NDNMulticastUDP6Address+"%"+localIf.Name+"]:"+strconv.Itoa(NDNMulticastUDPPort)), localURI, tlv.MaxNDNPacketSize)
	}
	// TODO: Get group address from config
	t.scope = ndn.NonLocal

	// Format group and local addresses
	t.groupAddr.IP = net.ParseIP(t.remoteURI.PathHost())
	t.groupAddr.Port = int(t.remoteURI.Port())
	t.groupAddr.Zone = t.remoteURI.PathZone()
	t.localAddr.IP = net.ParseIP(t.localURI.PathHost())
	t.localAddr.Port = 0 // int(t.localURI.Port())
	t.localAddr.Zone = t.localURI.PathZone()

	// Configure dialer so we can allow address reuse
	dialer := &net.Dialer{LocalAddr: &t.localAddr, Control: impl.SyscallReuseAddr}

	// Create send connection
	t.sendConn, err = dialer.Dial(t.remoteURI.Scheme(), t.groupAddr.String())
	if err != nil {
		return nil, errors.New("Unable to create send connection to group address: " + err.Error())
	}

	// Create receive connection
	t.recvConn, err = net.ListenMulticastUDP(t.remoteURI.Scheme(), localIf, &t.groupAddr)
	if err != nil {
		return nil, errors.New("Unable to create receive connection for group address on " + localIf.Name + ": " + err.Error())
	}

	t.changeState(ndn.Up)

	return t, nil
}

func (t *MulticastUDPTransport) String() string {
	return "MulticastUDPTransport, FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

func (t *MulticastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	_, err := t.sendConn.Write(frame)
	if err != nil {
		core.LogWarn("Unable to send on socket - DROP and Face DOWN")
		t.changeState(ndn.Down)
	}
}

func (t *MulticastUDPTransport) runReceive() {
	recvBuf := make([]byte, tlv.MaxNDNPacketSize)
	for {
		readSize, remoteAddr, err := t.recvConn.ReadFromUDP(recvBuf)
		if err != nil {
			if err.Error() == "EOF" {
				core.LogDebug(t, "EOF - Face DOWN")
			} else {
				core.LogWarn(t, "Unable to read from socket ("+err.Error()+") - DROP and Face DOWN")
			}
			t.changeState(ndn.Down)
			break
		}

		core.LogTrace(t, "Receive of size", readSize, "from", remoteAddr.String())

		if readSize > tlv.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
		}

		// Determine whether valid packet received
		_, _, tlvSize, err := tlv.DecodeTypeLength(recvBuf[:readSize])
		if err != nil {
			core.LogInfo("Unable to process received packet: " + err.Error())
		} else if readSize >= tlvSize {
			// Packet was successfully received, send up to link service
			t.linkService.handleIncomingFrame(recvBuf[:tlvSize])
		} else {
			core.LogInfo("Received packet is incomplete")
		}
	}
}

func (t *MulticastUDPTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "- state:", t.state, "->", new)
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing UDP socket")
		t.hasQuit <- true
		t.sendConn.Close()
		t.recvConn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
