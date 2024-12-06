/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/named-data/YaNFD/core"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

const lpPacketOverhead = 1 + 3
const pitTokenOverhead = 1 + 1 + 6
const congestionMarkOverhead = 3 + 1 + 8

const (
	FaceFlagLocalFields = 1 << iota
	FaceFlagLpReliabilityEnabled
	FaceFlagCongestionMarking
)

// NDNLPLinkServiceOptions contains the settings for an NDNLPLinkService.
type NDNLPLinkServiceOptions struct {
	IsFragmentationEnabled bool
	IsReassemblyEnabled    bool

	IsConsumerControlledForwardingEnabled bool

	IsIncomingFaceIndicationEnabled bool

	IsLocalCachePolicyEnabled bool

	IsCongestionMarkingEnabled bool

	BaseCongestionMarkingInterval   time.Duration
	DefaultCongestionThresholdBytes uint64
}

func MakeNDNLPLinkServiceOptions() NDNLPLinkServiceOptions {
	var o NDNLPLinkServiceOptions
	o.BaseCongestionMarkingInterval = time.Duration(100) * time.Millisecond
	o.DefaultCongestionThresholdBytes = uint64(math.Pow(2, 16))
	return o
}

// NDNLPLinkService is a link service implementing the NDNLPv2 link protocol
type NDNLPLinkService struct {
	linkServiceBase
	options        NDNLPLinkServiceOptions
	headerOverhead int

	// Receive
	partialMessageStore map[uint64][][]byte

	// Send
	nextSequence             uint64
	nextTxSequence           uint64
	lastTimeCongestionMarked time.Time
	BufferReader             enc.BufferReader
}

// MakeNDNLPLinkService creates a new NDNLPv2 link service
func MakeNDNLPLinkService(transport transport, options NDNLPLinkServiceOptions) *NDNLPLinkService {
	l := new(NDNLPLinkService)
	l.makeLinkServiceBase()
	l.transport = transport
	l.transport.setLinkService(l)
	l.options = options
	l.computeHeaderOverhead()

	l.partialMessageStore = make(map[uint64][][]byte)
	l.nextSequence = 0
	l.nextTxSequence = 0
	return l
}

func (l *NDNLPLinkService) String() string {
	if l.transport != nil {
		return "NDNLPLinkService, " + l.transport.String()
	}

	return "NDNLPLinkService, FaceID=" + strconv.FormatUint(l.faceID, 10)
}

// Options gets the settings of the NDNLPLinkService.
func (l *NDNLPLinkService) Options() NDNLPLinkServiceOptions {
	return l.options
}

// SetOptions changes the settings of the NDNLPLinkService.
func (l *NDNLPLinkService) SetOptions(options NDNLPLinkServiceOptions) {
	l.options = options
	l.computeHeaderOverhead()
}

func (l *NDNLPLinkService) computeHeaderOverhead() {
	l.headerOverhead = lpPacketOverhead // LpPacket (Type + Length of up to 2^16)

	if l.options.IsFragmentationEnabled {
		l.headerOverhead += 1 + 1 + 8 // Sequence
	}

	if l.options.IsFragmentationEnabled {
		l.headerOverhead += 1 + 1 + 2 + 1 + 1 + 2 // FragIndex/FragCount (Type + Length + up to 2^16 fragments)
	}

	if l.options.IsIncomingFaceIndicationEnabled {
		l.headerOverhead += 3 + 1 + 8 // IncomingFaceId
	}
}

// Run starts the face and associated goroutines
func (l *NDNLPLinkService) Run(optNewFrame []byte) {
	if l.transport == nil {
		core.LogError(l, "Unable to start face due to unset transport")
		return
	}

	if optNewFrame != nil {
		l.handleIncomingFrame(optNewFrame)
	}

	// Start transport goroutines
	go l.transport.runReceive()
	go l.runSend()

	// Wait for link service send goroutine to quit
	<-l.hasImplQuit
	l.HasQuit <- true
}
func sendPacket(l *NDNLPLinkService, netPacket *ndn_defn.PendingPacket) {
	wire := netPacket.RawBytes

	if l.transport.State() != ndn_defn.Up {
		core.LogWarn(l, "Attempted to send frame on down face - DROP and stop LinkService")
		l.hasImplQuit <- true
		return
	}
	// Counters
	if netPacket.EncPacket.Interest != nil {
		l.nOutInterests++
	} else if netPacket.EncPacket.Data != nil {
		l.nOutData++
	}

	now := time.Now()

	effectiveMtu := l.transport.MTU() - l.headerOverhead
	if netPacket.PitToken != nil {
		effectiveMtu -= pitTokenOverhead
	}
	if netPacket.CongestionMark != nil {
		effectiveMtu -= congestionMarkOverhead
	}

	// Fragmentation
	var fragments []*spec.LpPacket
	if int(wire.Length()) > effectiveMtu {
		if !l.options.IsFragmentationEnabled {
			core.LogInfo(l, "Attempted to send frame over MTU on link without fragmentation - DROP")
			return
		}

		// Split up fragment
		nFragments := int((wire.Length() + uint64(effectiveMtu) - 1) / uint64(effectiveMtu))
		lastFragSize := int(wire.Length()) - effectiveMtu*(nFragments-1)
		fragments = make([]*spec.LpPacket, nFragments)
		reader := enc.NewWireReader(wire)
		for i := 0; i < nFragments; i++ {
			if i < nFragments-1 {
				frag, err := reader.ReadWire(effectiveMtu)
				if err != nil {
					core.LogFatal(l, "Unexpected Wire reading error")
				}
				fragments[i] = &spec.LpPacket{
					Fragment: frag,
				}
			} else {
				frag, err := reader.ReadWire(lastFragSize)
				if err != nil {
					core.LogFatal(l, "Unexpected Wire reading error")
				}
				fragments[i] = &spec.LpPacket{
					Fragment: frag,
				}
			}
		}
	} else {
		fragments = make([]*spec.LpPacket, 1)
		fragments[0] = &spec.LpPacket{
			Fragment: wire,
		}
	}

	// Sequence
	if len(fragments) > 1 {
		for _, fragment := range fragments {
			fragment.Sequence = utils.IdPtr(l.nextSequence)
			l.nextSequence++
		}
	}

	// Congestion marking
	if congestionMarking && l.transport.GetSendQueueSize() > l.options.DefaultCongestionThresholdBytes &&
		now.After(l.lastTimeCongestionMarked.Add(l.options.BaseCongestionMarkingInterval)) {
		// Mark congestion
		core.LogWarn(l, "Marking congestion")
		fragments[0].CongestionMark = utils.IdPtr[uint64](1)
		l.lastTimeCongestionMarked = now
	}

	// PIT tokens
	if len(netPacket.PitToken) > 0 {
		fragments[0].PitToken = netPacket.PitToken
	}

	// Incoming face indication
	if l.options.IsIncomingFaceIndicationEnabled && netPacket.IncomingFaceID != nil {
		fragments[0].IncomingFaceId = netPacket.IncomingFaceID
	}

	// Congestion marking
	if netPacket.CongestionMark != nil {
		fragments[0].CongestionMark = netPacket.CongestionMark
	}

	// Send fragment(s)
	for _, fragment := range fragments {
		pkt := &spec.Packet{
			LpPacket: fragment,
		}
		encoder := spec.PacketEncoder{}
		encoder.Init(pkt)
		frameWire := encoder.Encode(pkt)
		if frameWire == nil {
			core.LogError(l, "Unable to encode fragment - DROP")
			break
		}
		l.transport.sendFrame(frameWire.Join())
	}
}
func (l *NDNLPLinkService) runSend() {
	core.LogTrace(l, "Starting send thread")

	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	for {
		select {
		case netPacket := <-l.sendQueue:
			sendPacket(l, netPacket)
		case <-l.hasTransportQuit:
			l.hasImplQuit <- true
			return
		}
	}
}

func (l *NDNLPLinkService) handleIncomingFrame(rawFrame []byte) {
	// We have to copy so receive transport buffer can be reused
	wire := make([]byte, len(rawFrame))
	copy(wire, rawFrame)
	l.processIncomingFrame(wire)
}

func (l *NDNLPLinkService) processIncomingFrame(wire []byte) {
	// all incoming frames come through a link service
	// Attempt to decode buffer into LpPacket
	netPacket := &ndn_defn.PendingPacket{
		IncomingFaceID: utils.IdPtr(l.faceID),
	}
	packet, _, e := spec.ReadPacket(enc.NewBufferReader(wire))
	if e != nil {
		core.LogError(l, e)
		return
	}
	if packet.LpPacket == nil {
		// Bare Data or Interest packet
		netPacket.RawBytes = enc.Wire{wire}
		netPacket.EncPacket = packet
	} else {
		fragment := packet.LpPacket.Fragment

		// If there is no fragment, then IDLE packet, drop.
		if len(fragment) == 0 {
			core.LogTrace(l, "IDLE frame - DROP")
			return
		}

		// Reassembly
		if l.options.IsReassemblyEnabled {
			if packet.LpPacket.Sequence == nil {
				core.LogInfo(l, "Received NDNLPv2 frame without Sequence but reassembly requires it - DROP")
				return
			}

			fragIndex := uint64(0)
			if packet.LpPacket.FragIndex != nil {
				fragIndex = *packet.LpPacket.FragIndex
			}
			fragCount := uint64(1)
			if packet.LpPacket.FragCount != nil {
				fragCount = *packet.LpPacket.FragCount
			}
			baseSequence := *packet.LpPacket.Sequence - fragIndex

			core.LogTrace(l, "Received fragment ", fragIndex, " of ", fragCount, " for ", baseSequence)
			if fragIndex == 0 && fragCount == 1 {
				// Bypass reassembly since only one fragment
			} else {
				fragment = l.reassemblePacket(packet.LpPacket, baseSequence, fragIndex, fragCount)
				if fragment == nil {
					// Nothing more to be done, so return
					return
				}
			}
		} else if packet.LpPacket.FragCount != nil || packet.LpPacket.FragIndex != nil {
			core.LogWarn(l, "Received NDNLPv2 frame containing fragmentation fields but reassembly disabled - DROP")
			return
		}

		// Congestion mark
		netPacket.CongestionMark = packet.LpPacket.CongestionMark

		// Consumer-controlled forwarding (NextHopFaceId)
		if l.options.IsConsumerControlledForwardingEnabled && packet.LpPacket.NextHopFaceId != nil {
			netPacket.NextHopFaceID = packet.LpPacket.NextHopFaceId
		}

		// Local cache policy
		if l.options.IsLocalCachePolicyEnabled && packet.LpPacket.CachePolicy != nil {
			netPacket.CachePolicy = &packet.LpPacket.CachePolicy.CachePolicyType
		}

		// PIT Token
		if len(packet.LpPacket.PitToken) > 0 {
			netPacket.PitToken = packet.LpPacket.PitToken
		}
		packet, _, e = spec.ReadPacket(enc.NewWireReader(fragment))
		if e != nil {
			return
		}
		netPacket.RawBytes = fragment
		netPacket.EncPacket = packet
	}
	// Counters
	if netPacket.EncPacket.Interest != nil {
		l.nInInterests++
	} else if netPacket.EncPacket.Data != nil {
		l.nInData++
	}
	l.dispatchIncomingPacket(netPacket)
}

func (l *NDNLPLinkService) reassemblePacket(
	frame *spec.LpPacket, baseSequence uint64, fragIndex uint64, fragCount uint64,
) enc.Wire {
	_, hasSequence := l.partialMessageStore[baseSequence]
	if !hasSequence {
		// Create map entry
		l.partialMessageStore[baseSequence] = make([][]byte, fragCount)
	}

	// Insert into PartialMessageStore
	// Safe to call Join since there is only one fragment
	if len(frame.Fragment) > 1 {
		core.LogError("LpPacket should only have one fragment.")
	}
	l.partialMessageStore[baseSequence][fragIndex] = frame.Fragment.Join()

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
		reassembled := make(enc.Wire, len(l.partialMessageStore[baseSequence]))
		for i, fragment := range l.partialMessageStore[baseSequence] {
			reassembled[i] = fragment
		}

		delete(l.partialMessageStore, baseSequence)
		return reassembled
	}

	return nil
}

func (op *NDNLPLinkServiceOptions) Flags() (ret uint64) {
	if op.IsConsumerControlledForwardingEnabled {
		ret |= FaceFlagLocalFields
	}
	if op.IsCongestionMarkingEnabled {
		ret |= FaceFlagCongestionMarking
	}
	return
}
