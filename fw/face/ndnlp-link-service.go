/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"container/list"
	"encoding/binary"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/fw"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/lpv2"
	"github.com/eric135/YaNFD/ndn/tlv"
)

const lpPacketOverhead = 1 + 3
const pitTokenOverhead = 1 + 1 + 2
const congestionMarkOverhead = 3 + 1 + 8
const ackOverhead = 3 + 1 + 8

// NDNLPLinkServiceOptions contains the settings for an NDNLPLinkService.
type NDNLPLinkServiceOptions struct {
	IsFragmentationEnabled bool

	IsReassemblyEnabled bool

	IsReliabilityEnabled bool
	MaxRetransmissions   int
	// ReservedAckSpace represents the number of Acks to reserve space for in
	ReservedAckSpace int

	IsConsumedControlledForwardingEnabled bool

	IsIncomingFaceIndicationEnabled bool

	IsLocalCachePolicyEnabled bool
}

type ndnlpUnacknowledgedFrame struct {
	frame              *lpv2.Packet
	netPacket          uint64 // Sequence of network-layer packet
	numRetransmissions int
	sentTime           time.Time
}

type ndnlpUnacknowledgedPacket struct {
	lock                    sync.Mutex             // Something being blocked on this should be an edge case
	unacknowledgedFragments map[uint64]interface{} // Unacknowledged fragments in packet
}

// NDNLPLinkService is a link service implementing the NDNLPv2 link protocol
type NDNLPLinkService struct {
	linkServiceBase
	options        NDNLPLinkServiceOptions
	headerOverhead int

	// Receive
	partialMessageStore map[uint64][][]byte
	pendingAcksToSend   *list.List
	idleAckTimer        chan interface{}

	// Send
	nextSequence          uint64
	unacknowledgedFrames  sync.Map    // TxSequence -> Frame
	unacknowledgedPackets sync.Map    // Sequence -> Network-layer packet
	retransmitQueue       chan uint64 // TxSequence
	rto                   time.Duration
	nextTxSequence        uint64
}

// MakeNDNLPLinkService creates a new NDNLPv2 link service
func MakeNDNLPLinkService(transport transport, options NDNLPLinkServiceOptions) *NDNLPLinkService {
	l := new(NDNLPLinkService)
	l.makeLinkServiceBase(transport)
	l.options = options
	l.computeHeaderOverhead()

	l.partialMessageStore = make(map[uint64][][]byte)
	l.pendingAcksToSend = list.New()
	l.idleAckTimer = make(chan interface{})

	l.nextSequence = 0
	l.retransmitQueue = make(chan uint64)
	l.rto = 0
	l.nextTxSequence = 0
	return l
}

// SetOptions changes the settings of the NDNLPLinkService.
func (l *NDNLPLinkService) SetOptions(options NDNLPLinkServiceOptions) {
	oldOptions := l.options
	l.options = options
	l.computeHeaderOverhead()

	if !oldOptions.IsReliabilityEnabled && l.options.IsReliabilityEnabled {
		go l.runRetransmit()
		go l.runIdleAckTimer()
	}
}

func (l *NDNLPLinkService) computeHeaderOverhead() {
	l.headerOverhead = lpPacketOverhead // LpPacket (Type + Length of up to 2^16)

	if l.options.IsFragmentationEnabled || l.options.IsReliabilityEnabled {
		l.headerOverhead += 1 + 1 + 8 // Sequence
	}

	if l.options.IsFragmentationEnabled {
		l.headerOverhead += 1 + 1 + 2 + 1 + 1 + 2 // FragIndex/FragCount (Type + Length + up to 2^16 fragments)
	}

	if l.options.IsReliabilityEnabled {
		l.headerOverhead += 3 + 1 + 8 // TxSequence
		l.headerOverhead += (3 + 1 + 8) * l.options.ReservedAckSpace
	}

	if l.options.IsIncomingFaceIndicationEnabled {
		l.headerOverhead += 3 + 1 + 8 // IncomingFaceId
	}
}

func (l *NDNLPLinkService) runSend() {
	if l.options.IsReliabilityEnabled {
		go l.runRetransmit()
		go l.runIdleAckTimer()
	}

	for !core.ShouldQuit {
		select {
		case netPacket := <-l.sendQueue:
			wire, err := netPacket.wire.Wire()
			if err != nil {
				core.LogWarn(l, "Unable to encode outgoing packet for queueing in link service - DROP")
				return
			}

			if l.transport.State() != Up {
				core.LogWarn(l, "Attempted to send frame on down face - DROP and stop LinkService")
				l.hasImplQuit <- true
				return
			}

			effectiveMtu := l.transport.MTU() - l.headerOverhead
			if netPacket.pitToken != nil {
				effectiveMtu -= pitTokenOverhead
			}
			if netPacket.congestionMark != nil {
				effectiveMtu -= congestionMarkOverhead
			}

			// Fragmentation
			fragments := []*lpv2.Packet{}
			if len(wire) > effectiveMtu {
				if !l.options.IsFragmentationEnabled {
					core.LogInfo(l, "Attempted to send frame over MTU on link without fragmentation - DROP")
					continue
				}

				// Split up fragment
				nFragments := int(math.Ceil(float64(len(wire)) / float64(effectiveMtu)))
				fragments = make([]*lpv2.Packet, nFragments)
				for i := 0; i < nFragments; i++ {
					if i < nFragments-1 {
						fragments[i] = lpv2.NewPacket(wire[effectiveMtu*i : effectiveMtu*(i+1)])
					} else {
						fragments[i] = lpv2.NewPacket(wire[effectiveMtu*i:])
					}
				}
			} else {
				fragments[0] = lpv2.NewPacket(wire)
			}

			// Sequence
			if len(fragments) > 0 || l.options.IsReliabilityEnabled {
				for _, fragment := range fragments {
					fragment.SetSequence(l.nextSequence)
					l.nextSequence++
				}
			}

			// Reliability
			if l.options.IsReliabilityEnabled {
				firstSequence := *fragments[0].Sequence()
				unacknowledgedPacket := new(ndnlpUnacknowledgedPacket)
				unacknowledgedPacket.lock.Lock()

				for _, fragment := range fragments {
					fragment.SetTxSequence(l.nextTxSequence)
					unacknowledgedFrame := new(ndnlpUnacknowledgedFrame)
					unacknowledgedFrame.frame = fragment
					unacknowledgedFrame.netPacket = firstSequence
					unacknowledgedFrame.sentTime = time.Now()
					l.unacknowledgedFrames.Store(l.nextTxSequence, unacknowledgedFrame)

					unacknowledgedPacket.unacknowledgedFragments[l.nextTxSequence] = new(interface{})
					l.nextTxSequence++
				}

				unacknowledgedPacket.lock.Unlock()
				l.unacknowledgedPackets.Store(firstSequence, unacknowledgedPacket)
			}

			// PIT tokens
			if netPacket.pitToken != nil {
				pitToken := make([]byte, 2)
				binary.BigEndian.PutUint16(pitToken, *netPacket.pitToken)
				fragments[0].SetPitToken(pitToken)
			}

			// Incoming face indication
			if netPacket.incomingFaceID != nil {
				fragments[0].SetIncomingFaceID(uint64(*netPacket.incomingFaceID))
			}

			// Congestion marking
			if netPacket.congestionMark != nil {
				fragments[0].SetCongestionMark(*netPacket.congestionMark)
			}

			// Fill up remaining space with Acks if Reliability enabled
			if l.options.IsReliabilityEnabled {
				// TODO
			}

			// Send fragment(s)
			for _, fragment := range fragments {
				encodedFragment, err := fragment.Encode()
				if err != nil {
					core.LogError(l, "Unable to encode fragment - DROP")
					break
				}
				fragmentWire, err := encodedFragment.Wire()
				if err != nil {
					core.LogError(l, "Unable to encode fragment - DROP")
					break
				}
				l.transport.sendFrame(fragmentWire)
			}
		case oldTxSequence := <-l.retransmitQueue:
			loadedFrame, ok := l.unacknowledgedFrames.Load(oldTxSequence)
			if !ok {
				// Frame must have been acknowledged between when noted as timed out and when processed here, so just silently ignore
				continue
			}
			frame := loadedFrame.(*ndnlpUnacknowledgedFrame)
			core.LogDebug(l, "Retransmitting TxSequence="+strconv.FormatUint(oldTxSequence, 10)+" of Sequence="+strconv.FormatUint(frame.netPacket, 10))
			// TODO
		case <-l.idleAckTimer:
			core.LogTrace(l, "Idle Ack timer expired")
			idle := new(lpv2.Packet)

			// Add up to enough Acks to fill MTU
			for remainingAcks := (l.transport.MTU() - lpPacketOverhead) / ackOverhead; remainingAcks > 0 && l.pendingAcksToSend.Len() > 0; remainingAcks-- {
				// TODO: Make pendingAcksToSend thread safe
				idle.AppendAck(l.pendingAcksToSend.Front().Value.(uint64))
				l.pendingAcksToSend.Remove(l.pendingAcksToSend.Front())
			}

			encoded, err := idle.Encode()
			if err != nil {
				core.LogError(l, "Unable to encode IDLE frame - DROP")
				break
			}
			wire, err := encoded.Wire()
			if err != nil {
				core.LogError(l, "Unable to encode IDLE frame - DROP")
				break
			}
			l.transport.sendFrame(wire)
		case <-l.hasTransportQuit:
			l.hasImplQuit <- true
			return
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

	core.LogDebug(l, "Received NDNLPv2 frame of size "+strconv.Itoa(len(rawFrame)))

	// Reliability
	if l.options.IsReliabilityEnabled {
		// Process Acks
		for _, ack := range frame.Acks() {
			if loadedAcknowledgedFrame, ok := l.unacknowledgedFrames.Load(ack); ok {
				core.LogTrace(l, "Received acknowledgement for TxSequence="+strconv.FormatUint(ack, 10))
				acknowledgedFrame := loadedAcknowledgedFrame.(*ndnlpUnacknowledgedFrame)
				sequence := acknowledgedFrame.netPacket
				loadedAcknowledgedPacket, _ := l.unacknowledgedPackets.Load(sequence)
				l.unacknowledgedFrames.Delete(ack)
				acknowledgedPacket := loadedAcknowledgedPacket.(*ndnlpUnacknowledgedPacket)
				acknowledgedPacket.lock.Lock()
				delete(acknowledgedPacket.unacknowledgedFragments, ack)
				remainingFragments := len(acknowledgedPacket.unacknowledgedFragments)
				acknowledgedPacket.lock.Unlock()
				if remainingFragments == 0 {
					core.LogTrace(l, "Completely transmitted reliable packet with Sequence="+strconv.FormatUint(sequence, 10))
					l.unacknowledgedPackets.Delete(sequence)
				}
			} else {
				core.LogDebug(l, "Received Ack for unknown TxSequence "+strconv.FormatUint(ack, 10))
			}
		}

		// Add TxSequence to Ack queue
		if frame.TxSequence() != nil {
			l.pendingAcksToSend.PushBack(*frame.TxSequence())
		}
	}

	// If no fragment, then IDLE frame, so DROP
	if frame.IsIdle() {
		core.LogTrace(l, "IDLE frame - DROP")
		return
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

	var netPacket PendingPacket
	netPacket.wire = netPkt

	// Congestion marking
	netPacket.congestionMark = frame.CongestionMark()

	// Consumer-controlled forwarding (NextHopFaceId)
	if l.options.IsConsumedControlledForwardingEnabled && frame.NextHopFaceID() != nil {
		netPacket.nextHopFaceID = frame.NextHopFaceID()
	}

	// Local cache policy
	if l.options.IsLocalCachePolicyEnabled && frame.CachePolicyType() != nil {
		netPacket.cachePolicy = frame.CachePolicyType()
	}

	// PIT Token
	if len(frame.PitToken()) == 2 {
		// YaNFD only sends PIT tokens as uint16, so implictly ignore other sizes.
		netPacket.pitToken = new(uint16)
		*netPacket.pitToken = binary.BigEndian.Uint16(frame.PitToken())
	}

	// Hand off to network layer by dispatching to appropriate forwarding thread(s)
	switch netPkt.Type() {
	case tlv.Interest:
		interest, err := ndn.DecodeInterest(netPkt)
		if err != nil {
			core.LogError(l, "Unable to decode Interest ("+err.Error()+") - DROP")
			break
		}
		thread := fw.HashNameToFwThread(interest.Name())
		core.LogTrace(l, "Dispatched Interest to thread "+strconv.Itoa(thread))
		fw.Threads[thread].QueueInterest(interest)
	case tlv.Data:
		data, err := ndn.DecodeData(netPkt, false)
		if err != nil {
			core.LogError(l, "Unable to decode Data ("+err.Error()+") - DROP")
			break
		}

		if netPacket.pitToken != nil {
			// If valid PIT token present, dispatch to that thread.
			if int(*netPacket.pitToken) >= len(fw.Threads) {
				// If invalid PIT token present, drop.
				core.LogError(l, "Invalid PIT token attached to Data packet - DROP")
				break
			}
			core.LogTrace(l, "Dispatched Interest to thread "+strconv.FormatUint(uint64(*netPacket.pitToken), 10))
			fw.Threads[int(*netPacket.pitToken)].QueueData(data)
		} else {
			// Otherwise, dispatch to threads matching every prefix.
			core.LogDebug(l, "Missing PIT token from Data packet - performing prefix dispatching")
			for _, thread := range fw.HashNameToAllPrefixFwThreads(data.Name()) {
				core.LogTrace(l, "Prefix dispatched Data packet to thread "+strconv.Itoa(thread))
				fw.Threads[thread].QueueData(data)
			}
		}
	default:
		core.LogError(l, "Cannot dispatch packet of unknown type "+strconv.FormatUint(uint64(netPkt.Type()), 10))
	}
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

func (l *NDNLPLinkService) removeUnacknowledgedPacket(sequence uint64) {
	if loadedUnacknowledgedPacket, ok := l.unacknowledgedPackets.Load(sequence); ok {
		unacknowledgedPacket := loadedUnacknowledgedPacket.(*ndnlpUnacknowledgedPacket)
		unacknowledgedPacket.lock.Lock()
		for txSequence := range unacknowledgedPacket.unacknowledgedFragments {
			l.unacknowledgedFrames.Delete(txSequence)
		}
		unacknowledgedPacket.lock.Unlock()
		l.unacknowledgedPackets.Delete(sequence)
	}
}

func (l *NDNLPLinkService) runRetransmit() {
	for l.options.IsReliabilityEnabled {
		curTime := time.Now()
		l.unacknowledgedFrames.Range(func(loadedTxSequence interface{}, loadedFrame interface{}) bool {
			txSequence := loadedTxSequence.(uint64)
			frame := loadedFrame.(*ndnlpUnacknowledgedFrame)
			if frame.sentTime.Add(l.rto).Before(curTime) {
				if frame.numRetransmissions >= l.options.MaxRetransmissions {
					// Drop entire network-layer packet because number of retransmissions exceeded
					core.LogDebug(l, "Network packet with Sequence number "+strconv.FormatUint(frame.netPacket, 10)+" exceeded allowed number of retransmissions - DROP")
					l.removeUnacknowledgedPacket(frame.netPacket)
				} else {
					// Indicate retransmission needed
					l.retransmitQueue <- txSequence
				}
			}
			return true
		})
		time.Sleep(5 * time.Millisecond)
	}
}

func (l *NDNLPLinkService) runIdleAckTimer() {
	for l.options.IsReliabilityEnabled {
		l.idleAckTimer <- new(interface{})
		time.Sleep(5 * time.Millisecond)
	}
}
