// +build !windows

/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"net"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// UnixStreamTransport is a Unix stream transport for communicating with local applications.
type UnixStreamTransport struct {
	conn net.Conn
	transportBase
}

// MakeUnixStreamTransport creates a Unix stream transport.
func MakeUnixStreamTransport(remoteURI URI, localURI URI, conn net.Conn) (*UnixStreamTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "fd" || !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	t := new(UnixStreamTransport)
	t.makeTransportBase(remoteURI, localURI, core.MaxNDNPacketSize)

	// Set scope and connection
	t.scope = Local
	t.conn = conn

	return t, nil
}

func (t *UnixStreamTransport) sendFrame(frame []byte) {
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

func (t *UnixStreamTransport) runReceive() {
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit && t.state != Down {
		readSize, err := t.conn.Read(recvBuf)
		if err != nil {
			core.LogWarn(t, "Unable to read from socket ("+err.Error()+") - DROP and Face DOWN")
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

func (t *UnixStreamTransport) onClose() {
	core.LogInfo(t, "Closing Unix stream socket")
	t.hasQuit <- true
	t.conn.Close()
}
