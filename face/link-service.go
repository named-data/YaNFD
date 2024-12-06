/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"encoding/binary"
	"strconv"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/fw"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
)

// LinkService is an interface for link service implementations
type LinkService interface {
	String() string
	Transport() transport
	SetFaceID(faceID uint64)

	FaceID() uint64
	LocalURI() *ndn_defn.URI
	RemoteURI() *ndn_defn.URI
	Persistency() Persistency
	SetPersistency(persistency Persistency)
	Scope() ndn_defn.Scope
	LinkType() ndn_defn.LinkType
	MTU() int
	SetMTU(mtu int)

	ExpirationPeriod() time.Duration
	State() ndn_defn.State

	// Run is the main entry point for running face thread
	// optNewFrame is optional new incoming frame
	Run(optNewFrame []byte)

	// SendPacket Add a packet to the send queue for this link service
	SendPacket(packet *ndn_defn.PendingPacket)
	handleIncomingFrame(frame []byte)

	Close()
	tellTransportQuit()
	GetHasQuit() chan bool

	// Counters
	NInInterests() uint64
	NInData() uint64
	NInBytes() uint64
	NOutInterests() uint64
	NOutData() uint64
	NOutBytes() uint64
}

// linkServiceBase is the type upon which all link service implementations should be built
type linkServiceBase struct {
	faceID           uint64
	transport        transport
	HasQuit          chan bool
	hasImplQuit      chan bool
	hasTransportQuit chan bool
	sendQueue        chan *ndn_defn.PendingPacket

	// Counters
	nInInterests  uint64
	nInData       uint64
	nOutInterests uint64
	nOutData      uint64
}

func (l *linkServiceBase) String() string {
	if l.transport != nil {
		return "LinkService, " + l.transport.String()
	}

	return "LinkService, FaceID=" + strconv.FormatUint(l.faceID, 10)
}

func (l *linkServiceBase) SetFaceID(faceID uint64) {
	l.faceID = faceID
	if l.transport != nil {
		l.transport.setFaceID(faceID)
	}
}

func (l *linkServiceBase) tellTransportQuit() {
	l.hasTransportQuit <- true
}

// GetHasQuit returns the channel that indicates when the face has quit.
func (l *linkServiceBase) GetHasQuit() chan bool {
	return l.HasQuit
}

//
// "Constructors" and threading
//

func (l *linkServiceBase) makeLinkServiceBase() {
	l.HasQuit = make(chan bool)
	l.hasImplQuit = make(chan bool)
	l.hasTransportQuit = make(chan bool)
	l.sendQueue = make(chan *ndn_defn.PendingPacket, faceQueueSize)
}

//
// Getters
//

// Transport returns the transport for the face.
func (l *linkServiceBase) Transport() transport {
	return l.transport
}

// FaceID returns the ID of the face
func (l *linkServiceBase) FaceID() uint64 {
	return l.faceID
}

// LocalURI returns the local URI of the underlying transport
func (l *linkServiceBase) LocalURI() *ndn_defn.URI {
	return l.transport.LocalURI()
}

// RemoteURI returns the remote URI of the underlying transport
func (l *linkServiceBase) RemoteURI() *ndn_defn.URI {
	return l.transport.RemoteURI()
}

// Persistency returns the MTU of the underlying transport.
func (l *linkServiceBase) Persistency() Persistency {
	return l.transport.Persistency()
}

// SetPersistency sets the MTU of the underlying transport.
func (l *linkServiceBase) SetPersistency(persistency Persistency) {
	l.transport.SetPersistency(persistency)
}

// Scope returns the scope of the underlying transport.
func (l *linkServiceBase) Scope() ndn_defn.Scope {
	return l.transport.Scope()
}

// LinkType returns the type of the link.
func (l *linkServiceBase) LinkType() ndn_defn.LinkType {
	return l.transport.LinkType()
}

// MTU returns the MTU of the underlying transport.
func (l *linkServiceBase) MTU() int {
	return l.transport.MTU()
}

// SetMTU sets the MTU of the underlying transport.
func (l *linkServiceBase) SetMTU(mtu int) {
	l.transport.SetMTU(mtu)
}

// ExpirationPeriod returns the time until the underlying transport expires. If transport not on-demand, returns 0.
func (l *linkServiceBase) ExpirationPeriod() time.Duration {
	return l.transport.ExpirationPeriod()
}

// State returns the state of the underlying transport.
func (l *linkServiceBase) State() ndn_defn.State {
	return l.transport.State()
}

//
// Counters
//

// NInInterests returns the number of Interests received on this face.
func (l *linkServiceBase) NInInterests() uint64 {
	return l.nInInterests
}

// NInData returns the number of Data packets received on this face.
func (l *linkServiceBase) NInData() uint64 {
	return l.nInData
}

// NInBytes returns the number of link-layer bytes received on this face.
func (l *linkServiceBase) NInBytes() uint64 {
	return l.transport.NInBytes()
}

// NOutInterests returns the number of Interests sent on this face.
func (l *linkServiceBase) NOutInterests() uint64 {
	return l.nOutInterests
}

// NInData returns the number of Data packets sent on this face.
func (l *linkServiceBase) NOutData() uint64 {
	return l.nOutData
}

// NOutBytes returns the number of link-layer bytes sent on this face.
func (l *linkServiceBase) NOutBytes() uint64 {
	return l.transport.NOutBytes()
}

//
// Forwarding pipeline
//

// SendPacket adds a packet to the send queue for this link service
func (l *linkServiceBase) SendPacket(packet *ndn_defn.PendingPacket) {
	select {
	case l.sendQueue <- packet:
		// Packet queued successfully
		core.LogTrace(l, "Queued packet for Link Service")
	default:
		// Drop packet due to congestion
		core.LogWarn(l, "Dropped packet due to congestion")

		// TODO: Signal congestion
	}
}

func (l *linkServiceBase) dispatchIncomingPacket(netPacket *ndn_defn.PendingPacket) {
	// Hand off to network layer by dispatching to appropriate forwarding thread(s)
	switch {
	case netPacket.EncPacket.Interest != nil:
		netPacket.NameCache = netPacket.EncPacket.Interest.NameV.String()
		thread := fw.HashNameToFwThread(netPacket.EncPacket.Interest.NameV)
		core.LogTrace(l, "Dispatched Interest to thread ", thread)
		dispatch.GetFWThread(thread).QueueInterest(netPacket)
	case netPacket.EncPacket.Data != nil:
		netPacket.NameCache = netPacket.EncPacket.Data.NameV.String()
		if len(netPacket.PitToken) == 6 {
			// Decode PitToken. If it's for us, it's a uint16 + uint32.
			pitTokenThread := binary.BigEndian.Uint16(netPacket.PitToken)
			fwThread := dispatch.GetFWThread(int(pitTokenThread))
			if fwThread == nil { // invalid PIT token present
				core.LogError(l, "Invalid PIT token attached to Data packet - DROP")
				break
			}

			core.LogTrace(l, "Dispatched Data to thread ", pitTokenThread)
			fwThread.QueueData(netPacket)
		} else if l.Scope() == ndn_defn.Local {
			// Only if from a local face (and therefore from a producer), dispatch to threads matching every prefix.
			// We need to do this because producers do not attach PIT tokens to their data packets.
			for _, thread := range fw.HashNameToAllPrefixFwThreads(netPacket.EncPacket.Data.NameV) {
				core.LogTrace(l, "Prefix dispatched local-origin Data packet to thread ", thread)
				dispatch.GetFWThread(thread).QueueData(netPacket)
			}
		} else {
			// Only exact-match for now (no CanBePrefix)
			thread := fw.HashNameToFwThread(netPacket.EncPacket.Data.NameV)
			core.LogTrace(l, "Dispatched Data to thread ", thread)
			dispatch.GetFWThread(thread).QueueData(netPacket)
		}
	default:
		core.LogError(l, "Cannot dispatch packet of unknown type ")
	}
}

func (l *linkServiceBase) Close() {
	l.transport.changeState(ndn_defn.Down)
}
