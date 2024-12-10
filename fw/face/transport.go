/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"time"

	defn "github.com/named-data/YaNFD/defn"
)

// transport provides an interface for transports for specific face types
type transport interface {
	String() string
	setFaceID(faceID uint64)
	setLinkService(linkService LinkService)

	RemoteURI() *defn.URI
	LocalURI() *defn.URI
	Persistency() Persistency
	SetPersistency(persistency Persistency) bool
	Scope() defn.Scope
	LinkType() defn.LinkType
	MTU() int
	SetMTU(mtu int)
	State() defn.State
	ExpirationPeriod() time.Duration

	GetSendQueueSize() uint64

	runReceive()

	sendFrame([]byte)

	changeState(newState defn.State)

	// Counters
	NInBytes() uint64
	NOutBytes() uint64
}

// transportBase provides logic common types between transport types
type transportBase struct {
	linkService LinkService

	faceID         uint64
	remoteURI      *defn.URI
	localURI       *defn.URI
	scope          defn.Scope
	persistency    Persistency
	linkType       defn.LinkType
	mtu            int
	expirationTime *time.Time

	state defn.State

	hasQuit chan bool

	// Counters
	nInBytes  uint64
	nOutBytes uint64
}

func (t *transportBase) makeTransportBase(remoteURI *defn.URI, localURI *defn.URI, persistency Persistency, scope defn.Scope, linkType defn.LinkType, mtu int) {
	t.remoteURI = remoteURI
	t.localURI = localURI
	t.persistency = persistency
	t.scope = scope
	t.linkType = linkType
	t.state = defn.Down
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

// LocalURI returns the local URI of the transport.
func (t *transportBase) LocalURI() *defn.URI {
	return t.localURI
}

// RemoteURI returns the remote URI of the transport.
func (t *transportBase) RemoteURI() *defn.URI {
	return t.remoteURI
}

// Persistency returns the persistency of the transport.
func (t *transportBase) Persistency() Persistency {
	return t.persistency
}

// Scope returns the scope of the transport.
func (t *transportBase) Scope() defn.Scope {
	return t.scope
}

// LinkType returns the type of the transport.
func (t *transportBase) LinkType() defn.LinkType {
	return t.linkType
}

// MTU returns the maximum transmission unit (MTU) of the Transport.
func (t *transportBase) MTU() int {
	return t.mtu
}

// SetMTU sets the MTU of the transport.
func (t *transportBase) SetMTU(mtu int) {
	t.mtu = mtu
}

// ExpirationPeriod returns the time until this face expires. If transport not on-demand, returns 0.
func (t *transportBase) ExpirationPeriod() time.Duration {
	if t.expirationTime == nil || t.persistency != PersistencyOnDemand {
		return 0
	}
	return time.Until(*t.expirationTime)
}

// State returns the state of the transport.
func (t *transportBase) State() defn.State {
	return t.state
}

//
// Counters
//

// NInBytes returns the number of link-layer bytes received on this transport.
func (t *transportBase) NInBytes() uint64 {
	return t.nInBytes
}

// NOutBytes returns the number of link-layer bytes sent on this transport.
func (t *transportBase) NOutBytes() uint64 {
	return t.nOutBytes
}

//
// Stubs
//

func (t *transportBase) runReceive() {
	// Overridden in specific transport implementation
}

func (t *transportBase) sendFrame(frame []byte) {
	// Overridden in specific transport implementation

	t.nOutBytes += uint64(len(frame))
}
