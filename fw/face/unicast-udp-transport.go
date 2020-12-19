/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
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
func MakeUnicastUDPTransport(remoteURI URI, localURI URI) (*UnicastUDPTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || (remoteURI.Scheme() != "udp4" && remoteURI.Scheme() != "udp6") || !localURI.IsCanonical() || remoteURI.Scheme() != localURI.Scheme() {
		return nil, core.ErrNotCanonical
	}

	t := new(UnicastUDPTransport)
	t.makeTransportBase(remoteURI, localURI, core.MaxNDNPacketSize)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = Local
	} else {
		t.scope = NonLocal
	}

	// Set local and remote addresses
	t.localAddr.IP = net.ParseIP(localURI.Path())
	t.localAddr.Port = int(localURI.Port())
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

	return t, nil
}

func (t *UnicastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	_, err := t.conn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP and Face DOWN")
		t.changeState(Down)
	}
}

func (t *UnicastUDPTransport) runReceive() {
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit && t.state != Down {
		readSize, err := t.conn.Read(recvBuf)
		if err != nil {
			core.LogWarn(t, "Unable to read from socket (", err, ") - DROP and Face DOWN")
			t.changeState(Down)
			break
		}

		core.LogTrace(t, "Receive of size", readSize)

		if readSize > core.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
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

	t.changeState(Down)
}

func (t *UnicastUDPTransport) onClose() {
	core.LogInfo(t, "Closing UDP socket")
	t.hasQuit <- true
	t.conn.Close()
}
