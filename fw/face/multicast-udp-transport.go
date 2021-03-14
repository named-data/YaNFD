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
	dialer    *net.Dialer
	sendConn  net.Conn
	recvConn  *net.UDPConn
	groupAddr net.UDPAddr
	localAddr net.UDPAddr
	isIPv4    bool
	transportBase
}

// MakeMulticastUDPTransport creates a new multicast UDP transport.
func MakeMulticastUDPTransport(localURI *ndn.URI) (*MulticastUDPTransport, error) {
	// Validate local URI
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	t := new(MulticastUDPTransport)
	// Get local interface
	localIf, err := InterfaceByIP(net.ParseIP(localURI.PathHost()))
	if err != nil || localIf == nil {
		core.LogError(t, "Unable to get interface for local URI "+localURI.String()+": "+err.Error())
	}

	if localURI.Scheme() == "udp4" {
		t.makeTransportBase(ndn.DecodeURIString("udp4://"+udp4MulticastAddress+":"+strconv.FormatUint(uint64(UDPMulticastPort), 10)), localURI, PersistencyPermanent, ndn.NonLocal, ndn.MultiAccess, tlv.MaxNDNPacketSize)
	} else if localURI.Scheme() == "udp6" {
		t.makeTransportBase(ndn.DecodeURIString("udp6://["+udp6MulticastAddress+"%"+localIf.Name+"]:"+strconv.FormatUint(uint64(UDPMulticastPort), 10)), localURI, PersistencyPermanent, ndn.NonLocal, ndn.MultiAccess, tlv.MaxNDNPacketSize)
	}
	t.scope = ndn.NonLocal

	// Format group and local addresses
	t.groupAddr.IP = net.ParseIP(t.remoteURI.PathHost())
	t.groupAddr.Port = int(t.remoteURI.Port())
	t.groupAddr.Zone = t.remoteURI.PathZone()
	t.localAddr.IP = net.ParseIP(t.localURI.PathHost())
	t.localAddr.Port = 0 // int(t.localURI.Port())
	t.localAddr.Zone = t.localURI.PathZone()

	// Configure dialer so we can allow address reuse
	t.dialer = &net.Dialer{LocalAddr: &t.localAddr, Control: impl.SyscallReuseAddr}

	// Create send connection
	t.sendConn, err = t.dialer.Dial(t.remoteURI.Scheme(), t.groupAddr.String())
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
	return "MulticastUDPTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *MulticastUDPTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	if persistency == PersistencyPermanent {
		t.persistency = persistency
		return true
	}

	return false
}

func (t *MulticastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size "+strconv.Itoa(len(frame)))
	_, err := t.sendConn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP")
		t.sendConn.Close()
		t.sendConn, err = t.dialer.Dial(t.remoteURI.Scheme(), t.groupAddr.String())
		if err != nil {
			core.LogError(t, "Unable to create send connection to group address: "+err.Error())
		}
	}
	t.nOutBytes += uint64(len(frame))
}

func (t *MulticastUDPTransport) runReceive() {
	recvBuf := make([]byte, tlv.MaxNDNPacketSize)
	for {
		readSize, remoteAddr, err := t.recvConn.ReadFromUDP(recvBuf)
		if err != nil {
			if err.Error() == "EOF" {
				core.LogDebug(t, "EOF - Face DOWN")
				t.changeState(ndn.Down)
				break
			} else {
				core.LogWarn(t, "Unable to read from socket ("+err.Error()+") - DROP")
				t.recvConn.Close()
				localIf, err := InterfaceByIP(net.ParseIP(t.localURI.PathHost()))
				if err != nil || localIf == nil {
					core.LogError(t, "Unable to get interface for local URI "+t.localURI.String()+": "+err.Error())
				}
				t.recvConn, _ = net.ListenMulticastUDP(t.remoteURI.Scheme(), localIf, &t.groupAddr)
			}
		}

		core.LogTrace(t, "Receive of size "+strconv.Itoa(readSize)+" from "+remoteAddr.String())
		t.nInBytes += uint64(readSize)

		if readSize > tlv.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
		}

		// Determine whether valid packet received
		_, _, tlvSize, err := tlv.DecodeTypeLength(recvBuf[:readSize])
		if err != nil {
			core.LogInfo(t, "Unable to process received packet: "+err.Error())
		} else if readSize >= tlvSize {
			// Packet was successfully received, send up to link service
			t.linkService.handleIncomingFrame(recvBuf[:tlvSize])
		} else {
			core.LogInfo(t, "Received packet is incomplete")
		}
	}
}

func (t *MulticastUDPTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: "+t.state.String()+" -> "+new.String())
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
