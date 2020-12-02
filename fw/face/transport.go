/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/go-ndn"
)

// Transport provides an interface for transports for specific face types
type transport interface {
	LocalURI() URI
	RemoteURI() URI
	State() State
	MTU() int

	RunReceive()
	RunSend()

	SendFrame([]byte)

	onClose()
}

// TransportBase provides logic common types between transport types
type transportBase struct {
	faceID    int
	remoteURI URI
	localURI  URI
	scope     Scope
	mtu       int

	state           State
	recvQueueForLS  chan ndn.LpPacket
	sendQueueFromLS chan []byte

	hasQuit chan bool
}

func newTransportBase(faceID int, remoteURI URI, localURI URI, mtu int) transportBase {
	return transportBase{
		faceID:          faceID,
		remoteURI:       remoteURI,
		localURI:        localURI,
		state:           Down,
		mtu:             mtu,
		recvQueueForLS:  make(chan ndn.LpPacket, core.FaceQueueSize),
		sendQueueFromLS: make(chan []byte, core.FaceQueueSize),
		hasQuit:         make(chan bool, 2)}
}

func (t *transportBase) String() string {
	return "(FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String() + ")"
}

//
// Getters
//

// LocalURI returns the local URI of the transport
func (t *transportBase) LocalURI() URI {
	return t.localURI
}

// RemoteURI returns the remote URI of the transport
func (t *transportBase) RemoteURI() URI {
	return t.remoteURI
}

// State returns the state of the transport
func (t *transportBase) State() State {
	return t.state
}

// MTU returns the maximum transmission unit (MTU) of the Transport
func (t *transportBase) MTU() int {
	return t.mtu
}

//
// Stubs
//

func (t *transportBase) RunReceive() {
	// Overridden in specific transport implementation
}

func (t *transportBase) RunSend() {
	// Overridden in specific transport implementation
}

func (t *transportBase) SendFrame(frame []byte) {
	// Overridden in specific transport implementation
}

func (t *transportBase) onClose() {
	// Overridden in specific transport implementation
}

//
// Helpers
//

func (t *transportBase) changeState(new State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "- state:", t.state, "->", new)
	t.state = new

	if t.state != Up {
		// Run implementation-specific close mechanisms
		t.onClose()

		// Send empty slice on link service channels to cause their goroutines to stop
		t.recvQueueForLS <- ndn.LpPacket{}
		t.sendQueueFromLS <- []byte{}
	}
}
