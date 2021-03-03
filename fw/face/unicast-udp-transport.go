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

// UnicastUDPTransport is a unicast UDP transport.
type UnicastUDPTransport struct {
	conn       net.Conn
	localAddr  net.UDPAddr
	remoteAddr net.UDPAddr
	transportBase
}

// MakeUnicastUDPTransport creates a new unicast UDP transport.
func MakeUnicastUDPTransport(remoteURI *ndn.URI, localURI *ndn.URI) (*UnicastUDPTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || (remoteURI.Scheme() != "udp4" && remoteURI.Scheme() != "udp6") || (localURI != nil && !localURI.IsCanonical()) || (localURI != nil && remoteURI.Scheme() != localURI.Scheme()) {
		return nil, core.ErrNotCanonical
	}

	t := new(UnicastUDPTransport)
	t.makeTransportBase(remoteURI, localURI, tlv.MaxNDNPacketSize)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = ndn.Local
	} else {
		t.scope = ndn.NonLocal
	}

	// Set local and remote addresses
	if localURI != nil {
		t.localAddr.IP = net.ParseIP(localURI.Path())
		t.localAddr.Port = int(localURI.Port())
	} else {
		t.localAddr.Port = int(UDPUnicastPort)
	}
	t.remoteAddr.IP = net.ParseIP(remoteURI.Path())
	t.remoteAddr.Port = int(remoteURI.Port())

	// Attempt to "dial" remote URI
	var err error
	// Configure dialer so we can allow address reuse
	dialer := &net.Dialer{LocalAddr: &t.localAddr, Control: impl.SyscallReuseAddr}
	t.conn, err = dialer.Dial(t.remoteURI.Scheme(), t.remoteURI.Path()+":"+strconv.Itoa(int(t.remoteURI.Port())))
	if err != nil {
		return nil, errors.New("Unable to connect to remote endpoint: " + err.Error())
	}

	if localURI == nil {
		t.localAddr = *t.conn.LocalAddr().(*net.UDPAddr)
		t.localURI = ndn.DecodeURIString("udp://" + t.localAddr.String())
	}

	t.changeState(ndn.Up)

	return t, nil
}

func (t *UnicastUDPTransport) String() string {
	return "UnicastUDPTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

func (t *UnicastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size "+strconv.Itoa(len(frame)))
	_, err := t.conn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP and Face DOWN")
		t.changeState(ndn.Down)
	}
	t.nOutBytes += uint64(len(frame))
}

func (t *UnicastUDPTransport) runReceive() {
	recvBuf := make([]byte, tlv.MaxNDNPacketSize)
	for {
		readSize, err := t.conn.Read(recvBuf)
		if err != nil {
			if err.Error() == "EOF" {
				core.LogDebug(t, "EOF - Face DOWN")
			} else {
				core.LogWarn(t, "Unable to read from socket ("+err.Error()+") - DROP and Face DOWN")
			}
			t.changeState(ndn.Down)
			break
		}

		core.LogTrace(t, "Receive of size "+strconv.Itoa(readSize))
		t.nInBytes += uint64(readSize)

		if readSize > tlv.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
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

func (t *UnicastUDPTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: "+t.state.String()+" -> "+new.String())
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing UDP socket")
		t.hasQuit <- true
		t.conn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
