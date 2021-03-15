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
	"time"

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
func MakeUnixStreamTransport(remoteURI *ndn.URI, localURI *ndn.URI, conn net.Conn) (*UnixStreamTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "fd" || !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	t := new(UnixStreamTransport)
	t.makeTransportBase(remoteURI, localURI, PersistencyOnDemand, ndn.Local, ndn.PointToPoint, tlv.MaxNDNPacketSize)
	t.expirationTime = new(time.Time)
	*t.expirationTime = time.Now().Add(udpLifetime)

	// Set connection
	t.conn = conn

	t.changeState(ndn.Up)

	go t.expirationHandler()

	return t, nil
}

func (t *UnixStreamTransport) String() string {
	return "UnixStreamTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *UnixStreamTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	if persistency == PersistencyOnDemand {
		t.persistency = persistency
		return true
	}

	return false
}

// expirationHandler checks if the face should expire (if on demand)
func (t *UnixStreamTransport) expirationHandler() {
	for {
		time.Sleep(time.Duration(10) * time.Second)
		if t.state == ndn.Down {
			break
		}
		if t.expirationTime.Before(time.Now()) || t.expirationTime.Equal(time.Now()) {
			core.LogInfo(t, "Face expired")
			t.changeState(ndn.Down)
			break
		}
	}
}

func (t *UnixStreamTransport) sendFrame(frame []byte) {
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

	*t.expirationTime = time.Now().Add(udpLifetime)
	t.nOutBytes += uint64(len(frame))
}

func (t *UnixStreamTransport) runReceive() {
	core.LogTrace(t, "Starting receive thread")
	recvBuf := make([]byte, tlv.MaxNDNPacketSize)
	for {
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

		core.LogTrace(t, "Receive of size "+strconv.Itoa(readSize))
		*t.expirationTime = time.Now().Add(udpLifetime)
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

func (t *UnixStreamTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: "+t.state.String()+" -> "+new.String())
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing Unix stream socket")
		t.hasQuit <- true
		t.conn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
