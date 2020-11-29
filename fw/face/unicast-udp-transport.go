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
	"github.com/eric135/go-ndn"
	"github.com/eric135/go-ndn/tlv"
)

// UnicastUDPTransport is a unicast UDP transport
type UnicastUDPTransport struct {
	conn       net.Conn
	recvBuf    [core.MaxNDNPacketSize]byte
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
	t := UnicastUDPTransport{transportBase: newTransportBase(faceID, remoteURI, URI{}, core.MaxNDNPacketSize)}
	var err error
	t.conn, err = net.Dial(remoteURI.Scheme(), remoteURI.Path()+":"+strconv.FormatUint(uint64(remoteURI.Port()), 10))
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// SendFrame adds a frame to the transport's send queue
func (t *UnicastUDPTransport) SendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	t.sendQueueFromLS <- frame
}

// RunReceive runs the receive goroutine
func (t *UnicastUDPTransport) RunReceive() {
	for !core.ShouldQuit {
		readSize, err := t.conn.Read(t.recvBuf[t.recvBufLen:])
		if err != nil {
			core.LogWarn(t, "Unable to read from socket - DROP and Face DOWN")
			t.conn.Close()
			t.changeState(Down)
			t.hasQuit <- true
			return
		}

		core.LogTrace(t, "Receive of size", readSize)

		t.recvBufLen += readSize

		// Attempt to decode packet from buffer
		var frame ndn.LpPacket
		err = tlv.Decode(t.recvBuf[:t.recvBufLen], &frame)
		if err == nil {
			// Sucessfully decoded a packet, send up to link service
			core.LogDebug(t, "Received frame of size", t.recvBufLen)
			t.recvQueueForLS <- frame

			// Erase from receive buffer
			t.recvBufLen = 0
		}
	}

	t.hasQuit <- true
}

// RunSend runs the send goroutine
func (t *UnicastUDPTransport) RunSend() {
	for !core.ShouldQuit {
		frame := <-t.sendQueueFromLS
		core.LogDebug(t, "Sending frame of size", len(frame))
		for nBytesToWrite := len(frame); nBytesToWrite > 0; {
			nBytesWritten, err := t.conn.Write(frame[len(frame)-nBytesToWrite:])
			if err != nil {
				core.LogWarn("Unable to write on socket - DROP and Face DOWN")
				t.conn.Close()
				t.changeState(Down)
				t.hasQuit <- true
				return
			}
			nBytesToWrite -= nBytesWritten
		}
	}

	t.hasQuit <- true
}
