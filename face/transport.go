/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// transport provides an interface for transports for specific face types
type transport interface {
	String() string
	setFaceID(faceID uint64)
	setLinkService(linkService LinkService)

	RemoteURI() *ndn.URI
	LocalURI() *ndn.URI
	Scope() ndn.Scope
	LinkType() ndn.LinkType
	MTU() int
	State() ndn.State

	runReceive()

	sendFrame([]byte)

	changeState(newState ndn.State)
}

// transportBase provides logic common types between transport types
type transportBase struct {
	linkService LinkService

	faceID    uint64
	remoteURI *ndn.URI
	localURI  *ndn.URI
	scope     ndn.Scope
	linkType  ndn.LinkType
	mtu       int

	state     ndn.State
	recvQueue chan *tlv.Block

	hasQuit chan bool
}

func (t *transportBase) makeTransportBase(remoteURI *ndn.URI, localURI *ndn.URI, mtu int) {
	t.remoteURI = remoteURI
	t.localURI = localURI
	t.state = ndn.Down
	t.mtu = mtu
	t.hasQuit = make(chan bool, 2)
}

func (t *transportBase) setFaceID(faceID uint64) {
	t.faceID = faceID
}

func (t *transportBase) setLinkService(linkService LinkService) {
	t.linkService = linkService
}

//
// Getters
//

// LocalURI returns the local URI of the transport
func (t *transportBase) LocalURI() *ndn.URI {
	return t.localURI
}

// RemoteURI returns the remote URI of the transport
func (t *transportBase) RemoteURI() *ndn.URI {
	return t.remoteURI
}

// Scope returns the scope of the transport
func (t *transportBase) Scope() ndn.Scope {
	return t.scope
}

// LinkType returns the type of the transport
func (t *transportBase) LinkType() ndn.LinkType {
	return t.linkType
}

// MTU returns the maximum transmission unit (MTU) of the Transport
func (t *transportBase) MTU() int {
	return t.mtu
}

// State returns the state of the transport
func (t *transportBase) State() ndn.State {
	return t.state
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

func (t *transportBase) receiveInitialFrameFromListener(frame []byte) {
	// Overridden in specific transport implementation
}
