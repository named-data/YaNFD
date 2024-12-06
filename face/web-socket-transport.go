/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"net"
	"runtime"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/named-data/YaNFD/core"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
)

// WebSocketTransport communicates with web applications via WebSocket.
type WebSocketTransport struct {
	transportBase
	c *websocket.Conn
}

var _ transport = &WebSocketTransport{}

// NewWebSocketTransport creates a Unix stream transport.
func NewWebSocketTransport(localURI *ndn_defn.URI, c *websocket.Conn) (t *WebSocketTransport) {
	remoteURI := ndn_defn.MakeWebSocketClientFaceURI(c.RemoteAddr())
	t = &WebSocketTransport{c: c}

	scope := ndn_defn.NonLocal
	ip := net.ParseIP(remoteURI.PathHost())
	if ip != nil && ip.IsLoopback() {
		scope = ndn_defn.Local
	}

	t.makeTransportBase(remoteURI, localURI, PersistencyOnDemand, scope, ndn_defn.PointToPoint, ndn_defn.MaxNDNPacketSize)
	t.changeState(ndn_defn.Up)
	return t
}

func (t *WebSocketTransport) String() string {
	return "WebSocketTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) +
		", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *WebSocketTransport) SetPersistency(persistency Persistency) bool {
	return persistency == PersistencyOnDemand
}

// GetSendQueueSize returns the current size of the send queue.
func (t *WebSocketTransport) GetSendQueueSize() uint64 {
	return 0
}

func (t *WebSocketTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size ", len(frame))
	e := t.c.WriteMessage(websocket.BinaryMessage, frame)
	if e != nil {
		core.LogWarn(t, "Unable to send on socket - DROP and Face DOWN")
		t.changeState(ndn_defn.Down)
	}

	t.nOutBytes += uint64(len(frame))
}

func (t *WebSocketTransport) runReceive() {
	core.LogTrace(t, "Starting receive thread")

	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	for {
		mt, message, e := t.c.ReadMessage()
		if e != nil {
			core.LogWarn(t, "Unable to read from socket (", e, ") - DROP and Face DOWN")
			t.changeState(ndn_defn.Down)
			break
		}

		if mt != websocket.BinaryMessage {
			core.LogWarn(t, "Ignored non-binary message")
			continue
		}

		core.LogTrace(t, "Receive of size ", len(message))
		t.nInBytes += uint64(len(message))

		if len(message) > ndn_defn.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
		}

		// Send up to link service
		t.linkService.handleIncomingFrame(message)
	}
}

func (t *WebSocketTransport) changeState(new ndn_defn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn_defn.Up {
		core.LogInfo(t, "Closing Unix stream socket")
		t.hasQuit <- true
		t.c.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
