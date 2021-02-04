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

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// UnixStreamTransport is a Unix stream transport for communicating with local applications.
type UnixStreamTransport struct {
	conn net.Conn
	transportBase
}

// MakeUnixStreamTransport creates a Unix stream transport.
func MakeUnixStreamTransport(remoteURI ndn.URI, localURI ndn.URI, conn net.Conn) (*UnixStreamTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "fd" || !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	t := new(UnixStreamTransport)
	t.makeTransportBase(remoteURI, localURI, tlv.MaxNDNPacketSize)

	// Set scope and connection
	t.scope = ndn.Local
	t.conn = conn

	t.changeState(ndn.Up)

	return t, nil
}

func (t *UnixStreamTransport) String() string {
	return "UnixStreamTransport, FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
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
		t.changeState(ndn.Down)
	}
}

func (t *UnixStreamTransport) runReceive() {
	core.LogTrace(t, "Starting receive thread")
	t.state = ndn.Up
	recvBuf := make([]byte, tlv.MaxNDNPacketSize)
	for !core.ShouldQuit && t.state != ndn.Down {
		core.LogTrace(t, "Reading from socket")
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

		core.LogTrace(t, "Receive of size", readSize)

		if readSize > tlv.MaxNDNPacketSize {
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

	t.changeState(ndn.Down)
}

func (t *UnixStreamTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "- state:", t.state, "->", new)
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing Unix stream socket")
		t.hasQuit <- true
		t.conn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()
	}
}
