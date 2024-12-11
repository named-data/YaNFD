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

	"github.com/named-data/YaNFD/core"
	defn "github.com/named-data/YaNFD/defn"
	"github.com/named-data/YaNFD/face/impl"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// UnixStreamTransport is a Unix stream transport for communicating with local applications.
type UnixStreamTransport struct {
	conn *net.UnixConn
	transportBase
}

// MakeUnixStreamTransport creates a Unix stream transport.
func MakeUnixStreamTransport(remoteURI *defn.URI, localURI *defn.URI, conn net.Conn) (*UnixStreamTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "fd" || !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	t := new(UnixStreamTransport)
	t.makeTransportBase(remoteURI, localURI, PersistencyPersistent, defn.Local, defn.PointToPoint, defn.MaxNDNPacketSize)

	// Set connection
	t.conn = conn.(*net.UnixConn)
	t.running.Store(true)

	return t, nil
}

func (t *UnixStreamTransport) String() string {
	return "UnixStreamTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) +
		", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *UnixStreamTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	if persistency == PersistencyPersistent {
		t.persistency = persistency
		return true
	}

	return false
}

// GetSendQueueSize returns the current size of the send queue.
func (t *UnixStreamTransport) GetSendQueueSize() uint64 {
	rawConn, err := t.conn.SyscallConn()
	if err != nil {
		core.LogWarn(t, "Unable to get raw connection to get socket length: ", err)
	}
	return impl.SyscallGetSocketSendQueueSize(rawConn)
}

func (t *UnixStreamTransport) sendFrame(frame []byte) {
	if !t.running.Load() {
		return
	}

	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	_, err := t.conn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP and Face DOWN")
		t.Close()
		return
	}

	t.nOutBytes += uint64(len(frame))
}

func (t *UnixStreamTransport) runReceive() {
	defer t.Close()

	recvBuf := make([]byte, defn.MaxNDNPacketSize*32)
	recvOff := 0
	tlvOff := 0

	for {
		readSize, err := t.conn.Read(recvBuf[recvOff:])
		recvOff += readSize
		if err != nil {
			core.LogWarn(t, "Unable to read from socket (", err, ") - Face DOWN")
			return
		}

		t.nInBytes += uint64(readSize)

		// Determine whether valid packet received
		for {
			rdr := enc.NewBufferReader(recvBuf[tlvOff:recvOff])

			typ, err := enc.ReadTLNum(rdr)
			if err != nil {
				// Probably incomplete packet
				break
			}

			len, err := enc.ReadTLNum(rdr)
			if err != nil {
				// Probably incomplete packet
				break
			}

			tlvSize := typ.EncodingLength() + len.EncodingLength() + int(len)

			if recvOff-tlvOff >= tlvSize {
				// Packet was successfully received, send up to link service
				t.linkService.handleIncomingFrame(recvBuf[tlvOff : tlvOff+tlvSize])
				tlvOff += tlvSize
			} else if recvOff-tlvOff > defn.MaxNDNPacketSize {
				// Invalid packet, something went wrong
				core.LogWarn(t, "Received too much data without valid TLV block")
				return
			} else {
				// Incomplete packet (for sure)
				break
			}
		}

		// If less than one packet space remains in buffer, shift to beginning
		if recvOff-tlvOff < defn.MaxNDNPacketSize {
			copy(recvBuf, recvBuf[tlvOff:recvOff])
			recvOff -= tlvOff
			tlvOff = 0
		}
	}

}

func (t *UnixStreamTransport) Close() {
	if t.running.Swap(false) {
		t.conn.Close()
	}
}
