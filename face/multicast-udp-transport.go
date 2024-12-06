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
	"runtime"
	"strconv"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face/impl"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
)

// MulticastUDPTransport is a multicast UDP transport.
type MulticastUDPTransport struct {
	dialer    *net.Dialer
	sendConn  *net.UDPConn
	recvConn  *net.UDPConn
	groupAddr net.UDPAddr
	localAddr net.UDPAddr
	transportBase
}

// MakeMulticastUDPTransport creates a new multicast UDP transport.
func MakeMulticastUDPTransport(localURI *ndn_defn.URI) (*MulticastUDPTransport, error) {
	// Validate local URI
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	t := new(MulticastUDPTransport)
	// Get local interface
	localIf, err := InterfaceByIP(net.ParseIP(localURI.PathHost()))
	if err != nil || localIf == nil {
		core.LogError(t, "Unable to get interface for local URI ", localURI, ": ", err)
	}

	if localURI.Scheme() == "udp4" {
		t.makeTransportBase(
			ndn_defn.DecodeURIString("udp4://"+udp4MulticastAddress+":"+strconv.FormatUint(uint64(UDPMulticastPort), 10)),
			localURI, PersistencyPermanent, ndn_defn.NonLocal, ndn_defn.MultiAccess, ndn_defn.MaxNDNPacketSize)
	} else if localURI.Scheme() == "udp6" {
		t.makeTransportBase(
			ndn_defn.DecodeURIString("udp6://["+udp6MulticastAddress+"%"+localIf.Name+"]:"+
				strconv.FormatUint(uint64(UDPMulticastPort), 10)),
			localURI, PersistencyPermanent, ndn_defn.NonLocal, ndn_defn.MultiAccess, ndn_defn.MaxNDNPacketSize)
	}
	t.scope = ndn_defn.NonLocal

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
	sendConn, err := t.dialer.Dial(t.remoteURI.Scheme(), t.groupAddr.String())
	if err != nil {
		return nil, errors.New("Unable to create send connection to group address: " + err.Error())
	}
	t.sendConn = sendConn.(*net.UDPConn)

	// Create receive connection
	t.recvConn, err = net.ListenMulticastUDP(t.remoteURI.Scheme(), localIf, &t.groupAddr)
	if err != nil {
		return nil, errors.New("Unable to create receive connection for group address on " +
			localIf.Name + ": " + err.Error())
	}

	t.changeState(ndn_defn.Up)

	return t, nil
}

func (t *MulticastUDPTransport) String() string {
	return "MulticastUDPTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) +
		", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
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

// GetSendQueueSize returns the current size of the send queue.
func (t *MulticastUDPTransport) GetSendQueueSize() uint64 {
	rawConn, err := t.recvConn.SyscallConn()
	if err != nil {
		core.LogWarn(t, "Unable to get raw connection to get socket length: ", err)
	}
	return impl.SyscallGetSocketSendQueueSize(rawConn)
}

func (t *MulticastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size ", len(frame))
	_, err := t.sendConn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP")
		t.sendConn.Close()
		sendConn, err := t.dialer.Dial(t.remoteURI.Scheme(), t.groupAddr.String())
		if err != nil {
			core.LogError(t, "Unable to create send connection to group address: ", err)
		}
		t.sendConn = sendConn.(*net.UDPConn)
	}
	t.nOutBytes += uint64(len(frame))
}

func (t *MulticastUDPTransport) runReceive() {
	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	recvBuf := make([]byte, ndn_defn.MaxNDNPacketSize)
	for {
		readSize, remoteAddr, err := t.recvConn.ReadFromUDP(recvBuf)
		if err != nil {
			if err.Error() == "EOF" {
				core.LogDebug(t, "EOF - Face DOWN")
				t.changeState(ndn_defn.Down)
				break
			} else {
				core.LogWarn(t, "Unable to read from socket (", err, ") - DROP")
				t.recvConn.Close()
				localIf, err := InterfaceByIP(net.ParseIP(t.localURI.PathHost()))
				if err != nil || localIf == nil {
					core.LogError(t, "Unable to get interface for local URI ", t.localURI, ": ", err)
				}
				t.recvConn, _ = net.ListenMulticastUDP(t.remoteURI.Scheme(), localIf, &t.groupAddr)
			}
		}

		core.LogTrace(t, "Receive of size ", readSize, " from ", remoteAddr)
		t.nInBytes += uint64(readSize)

		if readSize > ndn_defn.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
		}
		if readSize <= 0 {
			core.LogInfo(t, "Socket close.")
			continue
		}

		// Determine whether valid packet received
		_, _, tlvSize, err := ndn_defn.DecodeTypeLength(recvBuf[:readSize])
		if err != nil {
			core.LogInfo(t, "Unable to process received packet: ", err)
		} else if readSize >= tlvSize {
			// Packet was successfully received, send up to link service
			t.linkService.handleIncomingFrame(recvBuf[:tlvSize])
		} else {
			core.LogInfo(t, "Received packet is incomplete")
		}
	}
}

func (t *MulticastUDPTransport) changeState(new ndn_defn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn_defn.Up {
		core.LogInfo(t, "Closing UDP socket")
		t.hasQuit <- true
		t.sendConn.Close()
		t.recvConn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
