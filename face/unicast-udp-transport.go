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
	"github.com/eric135/YaNFD/util"
)

// UnicastUDPTransport is a unicast UDP transport
type UnicastUDPTransport struct {
	conn       net.Conn
	recvBuf    []byte
	recvBufLen int
	transportBase
}

// NewUnicastUDPTransport creates a new unicast UDP transport
func NewUnicastUDPTransport(faceID int, remoteURI URI) (*UnicastUDPTransport, error) {
	// Validate remote URI
	if !remoteURI.IsCanonical() {
		return nil, errors.New("URI could not be canonized")
	}

	// Attempt to connect to remote URI
	t := UnicastUDPTransport{
		recvBuf:       make([]byte, 0, core.MaxNDNPacketSize),
		transportBase: newTransportBase(faceID, remoteURI, URI{}, core.MaxNDNPacketSize)}
	var err error
	t.conn, err = net.Dial(remoteURI.Scheme(), remoteURI.Path()+":"+strconv.FormatUint(uint64(remoteURI.Port()), 10))
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (t *UnicastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	for nBytesToWrite := len(frame); nBytesToWrite > 0; {
		nBytesWritten, err := t.conn.Write(frame[len(frame)-nBytesToWrite:])
		if err != nil {
			core.LogWarn("Unable to write on socket - DROP and Face DOWN")
			t.changeState(Down)
			t.hasQuit <- true
			return
		}
		nBytesToWrite -= nBytesWritten
	}
}

func (t *UnicastUDPTransport) runReceive() {
	for !core.ShouldQuit {
		readSize, err := t.conn.Read(t.recvBuf[t.recvBufLen:])
		if err != nil {
			core.LogWarn(t, "Unable to read from socket (", err, ") - DROP and Face DOWN")
			t.changeState(Down)
			t.hasQuit <- true
			return
		}

		core.LogTrace(t, "Receive of size", readSize)

		t.recvBufLen += readSize

		if t.recvBufLen > core.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP and Face DOWN")
			t.changeState(Down)
			t.hasQuit <- true
			return
		}

		// Determine whether entire packet received
		tlvType, tlvLength, err := util.DecodeTypeLength(t.recvBuf[:t.recvBufLen])
		tlvSize := tlvType.Size() + tlvLength.Size() + int(tlvLength)
		if err == nil && t.recvBufLen >= tlvSize {
			// Packet should be successfully received, send up to link service
			t.linkService.handleIncomingFrame(t.recvBuf[:tlvSize])
			t.recvBuf = t.recvBuf[tlvSize:t.recvBufLen]
			t.recvBufLen = 0
		}
	}

	t.hasQuit <- true
}

func (t *UnicastUDPTransport) onClose() {
	core.LogInfo(t, "Closing UDP socket")
	t.conn.Close()
}
