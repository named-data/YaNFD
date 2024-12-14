/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"strconv"

	"github.com/named-data/YaNFD/core"
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

	return "NullLinkService, FaceID=" + strconv.FormatUint(l.faceID, 10)
}

// Run runs the NullLinkService.
func (l *NullLinkService) Run(initial []byte) {
	FaceTable.Add(l)
	go func() {
		l.transport.runReceive()
		FaceTable.Remove(l.transport.FaceID())
	}()
}

func (l *NullLinkService) handleIncomingFrame(frame []byte) {
	// Do nothing
	core.LogDebug(l, "Received frame on null link service - DROP")
}
