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
	"github.com/eric135/YaNFD/ndn/tlv"
)

// LinkService is an interface for link service implementations
type LinkService interface {
	String() string
	setFaceID(faceID int)
	tellTransportQuit()

	FaceID() int
	LocalURI() URI
	RemoteURI() URI
	State() State

	// Main entry point for running face thread
	Run()
	runSend()

	// SendPacket Add a packet to the send queue for this link service
	SendPacket(packet *PendingPacket)
	handleIncomingFrame(frame []byte)
}

// linkServiceBase is the type upon which all link service implementations should be built
type linkServiceBase struct {
	faceID           int
	transport        transport
	HasQuit          chan bool
	hasImplQuit      chan bool
	hasTransportQuit chan bool
	sendQueue        chan *PendingPacket
}

// PendingPacket represents a pending network-layer packet to be sent or recently received on the link, plus any associated metadata.
type PendingPacket struct {
	wire           *tlv.Block
	pitToken       *uint16
	congestionMark *uint64
	incomingFaceID *uint64
	nextHopFaceID  *uint64
	cachePolicy    *uint64
}

func (l *linkServiceBase) String() string {
	if l.transport != nil {
		return l.transport.String() + " LinkService"
	}

	return "FaceID=" + strconv.Itoa(l.faceID) + " LinkService"
}

func (l *linkServiceBase) setFaceID(faceID int) {
	l.faceID = faceID
	if l.transport != nil {
		l.transport.setFaceID(faceID)
	}
}

func (l *linkServiceBase) tellTransportQuit() {
	l.hasTransportQuit <- true
}

//
// "Constructors" and threading
//

func (l *linkServiceBase) makeLinkServiceBase(transport transport) {
	l.setTransport(transport)
	l.HasQuit = make(chan bool)
	l.hasImplQuit = make(chan bool)
	l.hasTransportQuit = make(chan bool)
	l.sendQueue = make(chan *PendingPacket, core.FaceQueueSize)
}

func (l *linkServiceBase) setTransport(transport transport) {
	if transport == nil {
		return
	}

	l.transport = transport
	l.transport.setLinkService(l)
}

// Run starts the face and associated goroutines
func (l *linkServiceBase) Run() {
	if l.transport == nil {
		core.LogError(l, "Unable to start face due to unset transport")
		return
	}

	// Start transport goroutines
	go l.transport.runReceive()
	go l.runSend()

	// Wait for link service send goroutine to quit
	<-l.hasImplQuit

	// Wait for transport receive goroutine to quit
	<-l.hasTransportQuit
}

func (l *linkServiceBase) runSend() {
	// Stub
}

//
// Getters
//

// FaceID returns the ID of the face
func (l *linkServiceBase) FaceID() int {
	return l.faceID
}

// LocalURI returns the local URI of underlying transport
func (l *linkServiceBase) LocalURI() URI {
	return l.transport.LocalURI()
}

// RemoteURI returns the remote URI of underlying transport
func (l *linkServiceBase) RemoteURI() URI {
	return l.transport.RemoteURI()
}

// State returns the state of underlying transport
func (l *linkServiceBase) State() State {
	return l.transport.State()
}

//
// Forwarding pipeline
//

// SendPacket adds a packet to the send queue for this link service
func (l *linkServiceBase) SendPacket(packet *PendingPacket) {
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

func (l *linkServiceBase) handleIncomingFrame(frame []byte) {
	// Stub
}
