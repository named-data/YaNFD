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

	"github.com/gorilla/websocket"
	"github.com/named-data/YaNFD/core"
	defn "github.com/named-data/YaNFD/defn"
)

// WebSocketTransport communicates with web applications via WebSocket.
type WebSocketTransport struct {
	transportBase
	c *websocket.Conn
}

var _ transport = &WebSocketTransport{}

// NewWebSocketTransport creates a Unix stream transport.
func NewWebSocketTransport(localURI *defn.URI, c *websocket.Conn) (t *WebSocketTransport) {
	remoteURI := defn.MakeWebSocketClientFaceURI(c.RemoteAddr())
	t = &WebSocketTransport{c: c}
	t.running.Store(true)

	scope := defn.NonLocal
	ip := net.ParseIP(remoteURI.PathHost())
	if ip != nil && ip.IsLoopback() {
		scope = defn.Local
	}

	t.makeTransportBase(remoteURI, localURI, PersistencyOnDemand, scope, defn.PointToPoint, defn.MaxNDNPacketSize)
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
		t.Close()
		return
	}

	t.nOutBytes += uint64(len(frame))
}

func (t *WebSocketTransport) runReceive() {
	for {
		mt, message, e := t.c.ReadMessage()
		if e != nil {
			core.LogWarn(t, "Unable to read from socket (", e, ") - DROP and Face DOWN")
			t.Close()
			return
		}

		if mt != websocket.BinaryMessage {
			core.LogWarn(t, "Ignored non-binary message")
			continue
		}

		core.LogTrace(t, "Receive of size ", len(message))
		t.nInBytes += uint64(len(message))

		if len(message) > defn.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
		}

		t.linkService.handleIncomingFrame(message)
	}
}

func (t *WebSocketTransport) Close() {
	t.c.Close()
}
