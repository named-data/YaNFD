/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"sync/atomic"
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
	ExpirationPeriod() time.Duration
	FaceID() uint64

	// Get the number of queued outgoing packets
	GetSendQueueSize() uint64
	// Send a frame (make if copy if necessary)
	sendFrame([]byte)
	// Receive frames in an infinite loop
	runReceive()
	// Transport is currently running (up)
	IsRunning() bool
	// Close the transport (runReceive should exit)
	Close()

	// Counters
	NInBytes() uint64
	NOutBytes() uint64
}

// transportBase provides logic common types between transport types
type transportBase struct {
	linkService LinkService
	running     atomic.Bool

	faceID         uint64
	remoteURI      *defn.URI
	localURI       *defn.URI
	scope          defn.Scope
	persistency    Persistency
	linkType       defn.LinkType
	mtu            int
	expirationTime *time.Time

	// Counters
	nInBytes  uint64
	nOutBytes uint64
}

func (t *transportBase) makeTransportBase(
	remoteURI *defn.URI,
	localURI *defn.URI,
	persistency Persistency,
	scope defn.Scope,
	linkType defn.LinkType,
	mtu int,
) {
	t.running = atomic.Bool{}
	t.remoteURI = remoteURI
	t.localURI = localURI
	t.persistency = persistency
	t.scope = scope
	t.linkType = linkType
	t.mtu = mtu
}

//
// Setters
//

func (t *transportBase) setFaceID(faceID uint64) {
	t.faceID = faceID
}

func (t *transportBase) setLinkService(linkService LinkService) {
	t.linkService = linkService
}

//
// Getters
//

func (t *transportBase) LocalURI() *defn.URI {
	return t.localURI
}

func (t *transportBase) RemoteURI() *defn.URI {
	return t.remoteURI
}

func (t *transportBase) Persistency() Persistency {
	return t.persistency
}

func (t *transportBase) Scope() defn.Scope {
	return t.scope
}

func (t *transportBase) LinkType() defn.LinkType {
	return t.linkType
}

func (t *transportBase) MTU() int {
	return t.mtu
}

func (t *transportBase) SetMTU(mtu int) {
	t.mtu = mtu
}

// ExpirationPeriod returns the time until this face expires.
// If transport not on-demand, returns 0.
func (t *transportBase) ExpirationPeriod() time.Duration {
	if t.expirationTime == nil || t.persistency != PersistencyOnDemand {
		return 0
	}
	return time.Until(*t.expirationTime)
}

func (t *transportBase) FaceID() uint64 {
	return t.faceID
}

func (t *transportBase) IsRunning() bool {
	return t.running.Load()
}

//
// Counters
//

func (t *transportBase) NInBytes() uint64 {
	return t.nInBytes
}

func (t *transportBase) NOutBytes() uint64 {
	return t.nOutBytes
}
