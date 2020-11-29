/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"github.com/eric135/YaNFD/core"
	"github.com/eric135/go-ndn"
)

// Threads contains all face threads
var Threads map[int]*LinkServiceBase

// LinkService is an interface for link service implementations
type LinkService interface {
	FaceID() int
	LocalURI() URI
	RemoteURI() URI
	State() State

	// Main entry point for running face thread
	Run()
	runReceive()
	runSend()

	// SendPacket Add a packet to the send queue for this link service
	SendPacket(packet *ndn.Packet)
}

// LinkServiceBase is the type upon which all link service implementations should be built
type LinkServiceBase struct {
	faceID         int
	transport      *transportBase
	HasQuit        chan bool
	hasImplQuit    chan bool
	sendQueueForLS chan []byte
}

func (l *LinkServiceBase) String() string {
	return l.transport.String() + " LinkService"
}

//
// "Constructors" and threading
//

func (l *LinkServiceBase) newLinkService(faceID int, transport *transportBase) {
	l.faceID = faceID
	l.transport = transport
	l.HasQuit = make(chan bool)
	l.sendQueueForLS = make(chan []byte, core.FaceQueueSize)
}

// Run starts the face and associated goroutines
func (l *LinkServiceBase) Run() {
	if l.transport == nil {
		core.LogError("Unable to start face", l.faceID, "due to unset transport")
		return
	}

	// Start transport goroutines
	go l.transport.RunReceive()
	go l.transport.RunSend()

	go l.runReceive()
	go l.runSend()

	// Wait for link service implementation goroutines to quit
	<-l.hasImplQuit
	<-l.hasImplQuit

	// Wait for transport goroutines to quit
	<-l.transport.hasQuit
	<-l.transport.hasQuit
}

func (l *LinkServiceBase) runReceive() {
	// Stub
}

func (l *LinkServiceBase) runSend() {
	// Stub
}

//
// Getters
//

// FaceID returns the ID of the face
func (l *LinkServiceBase) FaceID() int {
	return l.faceID
}

// LocalURI returns the local URI of underlying transport
func (l *LinkServiceBase) LocalURI() URI {
	return l.transport.LocalURI()
}

// RemoteURI returns the remote URI of underlying transport
func (l *LinkServiceBase) RemoteURI() URI {
	return l.transport.RemoteURI()
}

// State returns the state of underlying transport
func (l *LinkServiceBase) State() State {
	return l.transport.State()
}

//
// Forwarding pipeline
//

// SendPacket adds a packet to the send queue for this link service
func (l *LinkServiceBase) SendPacket(packet *ndn.Packet) {
	_, encoded, err := packet.MarshalTlv()
	if err != nil {
		core.LogWarn(l, "unable to encode outgoing packet for queueing in link service - DROP")
		return
	}

	if l.State() != Up {
		core.LogWarn(l, "cannot send packet on down face - DROP")
	}

	select {
	case l.sendQueueForLS <- encoded:
		// Packet queued successfully
		core.LogTrace(l, "queued packet for Link Service")
	default:
		// Drop packet due to congestion
		core.LogWarn(l, "dropped packet due to congestion")

		// TODO: Signal congestion
	}
}
