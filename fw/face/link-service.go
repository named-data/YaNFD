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

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/dispatch"
	"github.com/eric135/YaNFD/fw"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// LinkService is an interface for link service implementations
type LinkService interface {
	String() string
	Transport() transport
	SetFaceID(faceID uint64)

	FaceID() uint64
	LocalURI() *ndn.URI
	RemoteURI() *ndn.URI
	Scope() ndn.Scope
	LinkType() ndn.LinkType
	MTU() int
	SetMTU(mtu int)

	State() ndn.State

	// Main entry point for running face thread
	Run()

	// SendPacket Add a packet to the send queue for this link service
	SendPacket(packet *ndn.PendingPacket)
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
	sendQueue        chan *ndn.PendingPacket

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
	l.sendQueue = make(chan *ndn.PendingPacket, faceQueueSize)
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
func (l *linkServiceBase) LocalURI() *ndn.URI {
	return l.transport.LocalURI()
}

// RemoteURI returns the remote URI of the underlying transport
func (l *linkServiceBase) RemoteURI() *ndn.URI {
	return l.transport.RemoteURI()
}

// Scope returns the scope of the underlying transport
func (l *linkServiceBase) Scope() ndn.Scope {
	return l.transport.Scope()
}

// LinkType returns the type of the link.
func (l *linkServiceBase) LinkType() ndn.LinkType {
	return l.transport.LinkType()
}

// MTU returns the MTU of the underlying transport
func (l *linkServiceBase) MTU() int {
	return l.transport.MTU()
}

// SetMTU sets the MTU of the underlying transport.
func (l *linkServiceBase) SetMTU(mtu int) {
	l.transport.SetMTU(mtu)
}

// State returns the state of the underlying transport
func (l *linkServiceBase) State() ndn.State {
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
func (l *linkServiceBase) SendPacket(packet *ndn.PendingPacket) {
	/*if l.State() != Up {
		core.LogWarn(l, "Cannot send packet on down face - DROP")
	}*/

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

func (l *linkServiceBase) dispatchIncomingPacket(netPacket *ndn.PendingPacket) {
	// Hand off to network layer by dispatching to appropriate forwarding thread(s)
	switch netPacket.Wire.Type() {
	case tlv.Interest:
		interest, err := ndn.DecodeInterest(netPacket.Wire)
		if err != nil {
			core.LogError(l, "Unable to decode Interest ("+err.Error()+") - DROP")
			break
		}
		thread := fw.HashNameToFwThread(interest.Name())
		core.LogTrace(l, "Dispatched Interest to thread "+strconv.Itoa(thread))
		dispatch.GetFWThread(thread).QueueInterest(netPacket)
	case tlv.Data:
		if len(netPacket.PitToken) == 6 {
			// Decode PitToken. If it's for us, it's a uint16 + uint32.
			pitTokenThread := binary.BigEndian.Uint16(netPacket.PitToken)
			fwThread := dispatch.GetFWThread(int(pitTokenThread))
			if fwThread == nil {
				// If invalid PIT token present, drop.
				core.LogError(l, "Invalid PIT token attached to Data packet - DROP")
				break
			}
			// If valid PIT token present, dispatch to that thread.
			core.LogTrace(l, "Dispatched Interest to thread "+strconv.FormatUint(uint64(pitTokenThread), 10))
			fwThread.QueueData(netPacket)
		} else {
			// Otherwise, dispatch to threads matching every prefix.
			core.LogDebug(l, "Missing or invalid PIT token in Data packet - DROP")
		}
	default:
		core.LogError(l, "Cannot dispatch packet of unknown type "+strconv.FormatUint(uint64(netPacket.Wire.Type()), 10))
	}
}

func (l *linkServiceBase) Close() {
	l.transport.changeState(ndn.Down)
}
