/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import "github.com/eric135/YaNFD/core"

type ndnlpLinkServiceOptions struct {
	IsFragmentationEnabled bool
	IsReassemblyEnabled    bool
	IsReliabilityEnabled   bool
}

// NDNLPLinkService is a link service implementing the NDNLPv2 link protocol
type NDNLPLinkService struct {
	options ndnlpLinkServiceOptions
	LinkServiceBase
}

// NewNDNLPLinkService creates a new NDNLPv2 link service
func NewNDNLPLinkService(faceID int, transport *transportBase) NDNLPLinkService {
	l := NDNLPLinkService{options: ndnlpLinkServiceOptions{true, true, false}}
	l.newLinkService(faceID, transport)
	return l
}

func (l *NDNLPLinkService) runReceive() {
	for !core.ShouldQuit {
		frame := <-l.transport.recvQueueForLS
		if l.transport.State() != Up {
			core.LogWarn(l, "- attempting to receive frame on down face")
			l.hasImplQuit <- true
			return
		} else if l.transport.State() == AdminDown {
			core.LogWarn(l, "- attempting to receive frame on admin down face")
		}

		// TODO: Do NDNLP things

		core.LogTrace(frame)

		// Hash to forwarding thread and place in queue
		// TODO
	}

	l.hasImplQuit <- true
}

func (l *NDNLPLinkService) runSend() {
	for !core.ShouldQuit {
		netPacket := <-l.sendQueueForLS
		if l.transport.State() != Up {
			core.LogWarn(l, "- attempting to send frame on down face - DROP and stop LinkService")
			l.hasImplQuit <- true
			return
		}

		// TODO: Do NDNLP things

		select {
		case l.transport.sendQueueFromLS <- netPacket:
			// Passed off to transport
		default:
			// Drop packet due to congestion
			core.LogWarn(l, "dropped packet due to congestion")

			// TODO: Signal congestion
		}
	}

	l.hasImplQuit <- true
}
