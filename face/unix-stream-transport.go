/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"bufio"
	"io"
	"net"
	"runtime"
	"strconv"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face/impl"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
)

// UnixStreamTransport is a Unix stream transport for communicating with local applications.
type UnixStreamTransport struct {
	conn   *net.UnixConn
	reader *bufio.Reader
	transportBase
}

// MakeUnixStreamTransport creates a Unix stream transport.
func MakeUnixStreamTransport(remoteURI *ndn_defn.URI, localURI *ndn_defn.URI, conn net.Conn) (*UnixStreamTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || remoteURI.Scheme() != "fd" || !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	t := new(UnixStreamTransport)
	t.makeTransportBase(remoteURI, localURI, PersistencyPersistent, ndn_defn.Local, ndn_defn.PointToPoint, ndn_defn.MaxNDNPacketSize)

	// Set connection
	t.conn = conn.(*net.UnixConn)
	t.reader = bufio.NewReaderSize(t.conn, 32*ndn_defn.MaxNDNPacketSize)

	t.changeState(ndn_defn.Up)

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
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size ", len(frame))
	_, err := t.conn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP and Face DOWN")
		t.changeState(ndn_defn.Down)
	}

	t.nOutBytes += uint64(len(frame))
}

func (t *UnixStreamTransport) runReceive() {
	core.LogTrace(t, "Starting receive thread")

	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	handleError := func(err error) {
		if err.Error() == "EOF" {
			core.LogDebug(t, "EOF - Face DOWN")
		} else {
			core.LogWarn(t, "Unable to read from socket (", err, ") - DROP and Face DOWN")
		}
		t.changeState(ndn_defn.Down)
	}

	recvBuf := make([]byte, ndn_defn.MaxNDNPacketSize)
	for {
		typ, len, err := ndn_defn.ReadTypeLength(t.reader)
		if err != nil {
			handleError(err)
			break
		}

		cursor := 0
		cursor += typ.EncodeInto(recvBuf[cursor:])
		cursor += len.EncodeInto(recvBuf[cursor:])

		lenRead, err := io.ReadFull(t.reader, recvBuf[cursor:cursor+int(len)])
		if err != nil {
			handleError(err)
			break
		}
		cursor += lenRead

		t.linkService.handleIncomingFrame(recvBuf[:cursor])
		t.nInBytes += uint64(cursor)
	}
}

func (t *UnixStreamTransport) changeState(new ndn_defn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn_defn.Up {
		core.LogInfo(t, "Closing Unix stream socket")
		t.hasQuit <- true
		t.conn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
