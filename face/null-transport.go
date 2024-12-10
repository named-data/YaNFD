/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"strconv"

	"github.com/named-data/YaNFD/core"
	defn "github.com/named-data/YaNFD/defn"
)

// NullTransport is a transport that drops all packets.
type NullTransport struct {
	transportBase
}

// MakeNullTransport makes a NullTransport.
func MakeNullTransport() *NullTransport {
	t := new(NullTransport)
	t.makeTransportBase(defn.MakeNullFaceURI(), defn.MakeNullFaceURI(), PersistencyPermanent, defn.NonLocal, defn.PointToPoint, defn.MaxNDNPacketSize)
	t.changeState(defn.Up)
	return t
}

func (t *NullTransport) String() string {
	return "NullTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *NullTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	if persistency == PersistencyPermanent {
		t.persistency = persistency
		return true
	}

	return false
}

// GetSendQueueSize returns the current size of the send queue.
func (t *NullTransport) GetSendQueueSize() uint64 {
	return 0
}

func (t *NullTransport) changeState(new defn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != defn.Up {
		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
