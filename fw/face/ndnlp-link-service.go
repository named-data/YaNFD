/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"math"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn/lpv2"
	"github.com/eric135/YaNFD/ndn/tlv"
)

type ndnlpLinkServiceOptions struct {
	IsFragmentationEnabled bool
	IsReassemblyEnabled    bool
	IsReliabilityEnabled   bool
}

// NDNLPLinkService is a link service implementing the NDNLPv2 link protocol
type NDNLPLinkService struct {
	options             ndnlpLinkServiceOptions
	partialMessageStore map[uint64][][]byte
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
			core.LogWarn(l, "Attempted to send frame on down face - DROP and stop LinkService")
			l.hasImplQuit <- true
			return
		}

		// Fragmentation
		fragments := []*lpv2.Packet{}
		if len(l.sendQueue) > l.transport.MTU() {
			if !l.options.IsFragmentationEnabled {
				core.LogInfo(l, "Attempted to send frame over MTU on link without fragmentation - DROP")
			}

			// Split up fragment
			nFragments := int(math.Ceil(float64(len(netPacket)) / float64(l.transport.MTU())))
			fragments = make([]*lpv2.Packet, nFragments)
			for i := 0; i < nFragments; i++ {
				if i < nFragments-1 {
					fragments[i] = lpv2.NewPacket(netPacket[l.transport.MTU()*i : l.transport.MTU()*(i+1)])
				} else {
					fragments[i] = lpv2.NewPacket(netPacket[l.transport.MTU()*i:])
				}
			}
		} else {
			fragments[0] = lpv2.NewPacket(netPacket)
		}

		// TODO: Reliability

		// TODO: PIT tokens

		// TODO: Incoming face indication

		// TODO: Congestion marking

		if len(fragments) == 0 {
			l.transport.sendFrame(netPacket)
		} else {

		}
	}

	l.hasImplQuit <- true
}

func (l *NDNLPLinkService) handleIncomingFrame(rawFrame []byte) {
	// Attempt to decode buffer into TLV block
	block, _, err := tlv.DecodeBlock(rawFrame)

	// Now attempt to decode LpPacket from block
	frame, err := lpv2.DecodePacket(block)
	if err != nil {
		core.LogDebug(l, "Received invalid frame - DROP")
	}

	core.LogDebug(l, "Received NDNLPv2 frame of size", len(rawFrame))

	// Reliability
	if l.options.IsReliabilityEnabled {
		// Process Acks
		// TODO

		// Add TxSequence to Ack queue
		// TODO
	}

	// Reassembly
	netPkt := frame.Fragment()
	if l.options.IsReassemblyEnabled && frame.Fragment() != nil {
		if frame.Sequence() == nil {
			core.LogInfo(l, "Received NDNLPv2 frame without Sequence but reassembly requires it - DROP")
			return
		}

		fragIndex := uint64(0)
		if frame.FragIndex() != nil {
			fragIndex = *frame.FragIndex()
		}
		fragCount := uint64(1)
		if frame.FragCount() != nil {
			fragCount = *frame.FragCount()
		}
		baseSequence := *frame.Sequence() - fragIndex

		core.LogDebug(l, "Received fragment", fragIndex, "of", fragCount, "for", baseSequence)

		if fragIndex == 0 && fragCount == 1 {
			// Bypass reassembly since only one fragment
		} else {
			netPkt = l.reassemblePacket(frame, baseSequence, fragIndex, fragCount)
			if netPkt == nil {
				// Nothing more to be done, so return
				return
			}
		}
	} else if frame.FragCount() != nil || frame.FragIndex() != nil {
		core.LogWarn(l, "Received NDNLPv2 frame containing fragmentation fields but reassembly disabled - DROP")
		return
	}

	// Hand off to network layer
	// Which will hash to forwarding thread and place in queue based upon type
	// TODO
}

func (l *NDNLPLinkService) reassemblePacket(frame *lpv2.Packet, baseSequence uint64, fragIndex uint64, fragCount uint64) *tlv.Block {
	_, hasSequence := l.partialMessageStore[baseSequence]
	if !hasSequence {
		// Create map entry
		l.partialMessageStore[baseSequence] = make([][]byte, fragCount)
	}

	// Insert into PartialMessageStore
	l.partialMessageStore[baseSequence][fragIndex] = make([]byte, len(frame.Fragment().Value()))
	copy(l.partialMessageStore[baseSequence][fragIndex], frame.Fragment().Value())

	// Determine whether it is time to reassemble
	receivedCount := 0
	receivedTotalLen := 0
	for _, fragment := range l.partialMessageStore[baseSequence] {
		if len(fragment) != 0 {
			receivedCount++
			receivedTotalLen += len(fragment)
		}
	}

	if receivedCount == len(l.partialMessageStore[baseSequence]) {
		// Time to reassemble!
		reassembled := new(tlv.Block)
		reassembled.SetType(lpv2.Fragment)

		reassembledValue := make([]byte, receivedTotalLen)
		reassembledSize := 0
		for _, fragment := range l.partialMessageStore[baseSequence] {
			copy(reassembledValue[reassembledSize:], fragment)
		}
		reassembled.SetValue(reassembledValue)

		delete(l.partialMessageStore, baseSequence)
		return reassembled
	}

	return nil
}
