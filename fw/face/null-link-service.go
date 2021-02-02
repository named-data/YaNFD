/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"strconv"

	"github.com/eric135/YaNFD/core"
)

// NullLinkService is a link service that drops all packets.
type NullLinkService struct {
	linkServiceBase
}

// MakeNullLinkService makes a NullLinkService.
func MakeNullLinkService(transport transport) *NullLinkService {
	l := new(NullLinkService)
	l.makeLinkServiceBase()
	l.transport = transport
	l.transport.setLinkService(l)
	return l
}

func (l *NullLinkService) String() string {
	if l.transport != nil {
		return "NullLinkService, " + l.transport.String()
	}

	return "NullLinkService, FaceID=" + strconv.Itoa(l.faceID)
}

// Run runs the NullLinkService.
func (l *NullLinkService) Run() {
	if l.transport == nil {
		core.LogError(l, "Unable to start face due to unset transport")
		return
	}

	// Start transport goroutines
	go l.transport.runReceive()

	// Wait for transport receive goroutine to quit
	<-l.hasTransportQuit

	core.LogTrace(l, "Transport has quit")

	l.HasQuit <- true
}

func (l *NullLinkService) handleIncomingFrame(frame []byte) {
	// Do nothing
	core.LogDebug(l, "Received frame on null link service - DROP")
}
