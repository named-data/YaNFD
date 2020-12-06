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
	"github.com/eric135/go-ndn/tlv"
)

type ndnlpLinkServiceOptions struct {
	IsFragmentationEnabled bool
	IsReassemblyEnabled    bool
	IsReliabilityEnabled   bool
}

// NDNLPLinkService is a link service implementing the NDNLPv2 link protocol
type NDNLPLinkService struct {
	options ndnlpLinkServiceOptions
	linkServiceBase
}

// MakeNDNLPLinkService creates a new NDNLPv2 link service
func MakeNDNLPLinkService(transport transport) *NDNLPLinkService {
	l := NDNLPLinkService{options: ndnlpLinkServiceOptions{true, true, false}}
	l.makeLinkServiceBase(transport)
	return &l
}

func (l *NDNLPLinkService) runSend() {
	var netPacket []byte
	for !core.ShouldQuit {
		select {
		case netPacket = <-l.sendQueue:
		case <-l.hasTransportQuit:
			l.hasImplQuit <- true
			return
		}

		if l.transport.State() != Up {
			core.LogWarn(l, "- attempting to send frame on down face - DROP and stop LinkService")
			l.hasImplQuit <- true
			return
		}

		// TODO: Do NDNLP things

		l.transport.sendFrame(netPacket)
	}

	l.hasImplQuit <- true
}

func (l *NDNLPLinkService) handleIncomingFrame(rawFrame []byte) {
	// Attempt to decode frame from buffer
	var frame ndn.LpPacket
	err := tlv.Decode(rawFrame, &frame)
	if err != nil {
		core.LogDebug(l, "Received invalid frame - DROP")
	}

	core.LogDebug(l, "Received NDNLPv2 frame of size", len(rawFrame))

	// TODO: Do NDNLP things

	// Hash to forwarding thread and place in queue
	// TODO
}
