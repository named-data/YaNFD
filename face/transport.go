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

// transport provides an interface for transports for specific face types
type transport interface {
	String() string
	setLinkService(linkService LinkService)

	LocalURI() URI
	RemoteURI() URI
	State() State
	MTU() int

	runReceive()

	sendFrame([]byte)

	onClose()
}

// transportBase provides logic common types between transport types
type transportBase struct {
	linkService LinkService

	faceID    int
	remoteURI URI
	localURI  URI
	scope     Scope
	mtu       int

	state     State
	recvQueue chan ndn.LpPacket

	hasQuit chan bool
}

func newTransportBase(faceID int, remoteURI URI, localURI URI, mtu int) transportBase {
	return transportBase{
		faceID:    faceID,
		remoteURI: remoteURI,
		localURI:  localURI,
		state:     Down,
		mtu:       mtu,
		hasQuit:   make(chan bool, 2)}
}

func (t *transportBase) String() string {
	return "(FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String() + ")"
}

func (t *transportBase) setLinkService(linkService LinkService) {
	t.linkService = linkService
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

func (t *transportBase) runReceive() {
	// Overridden in specific transport implementation
}

func (t *transportBase) sendFrame(frame []byte) {
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

		// Stop link service
		t.linkService.tellTransportQuit()
	}
}
