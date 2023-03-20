/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"container/list"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/lpv2"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	"github.com/zjkmxy/stealthpool"
)

const lpPacketOverhead = 1 + 3
const pitTokenOverhead = 1 + 1 + 6
const congestionMarkOverhead = 3 + 1 + 8
const ackOverhead = 3 + 1 + 8

const maxPoolBlockCnt = 1000
const maxPoolBlockSize = 9000
const workers = 16

const (
	FaceFlagLocalFields = 1 << iota
	FaceFlagLpReliabilityEnabled
	FaceFlagCongestionMarking
)

// NDNLPLinkServiceOptions contains the settings for an NDNLPLinkService.
type NDNLPLinkServiceOptions struct {
	IsFragmentationEnabled bool

	IsReassemblyEnabled bool

	IsReliabilityEnabled bool
	MaxRetransmissions   int
	// ReservedAckSpace represents the number of Acks to reserve space for in
	ReservedAckSpace int

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
	nextSequence             uint64
	unacknowledgedFrames     sync.Map    // TxSequence -> Frame
	unacknowledgedPackets    sync.Map    // Sequence -> Network-layer packet
	retransmitQueue          chan uint64 // TxSequence
	rto                      time.Duration
	nextTxSequence           uint64
	lastTimeCongestionMarked time.Time
	bees                     []*WorkerBee
	workQueue                chan *ndn.PendingPacket
	stealthPool              *stealthpool.Pool
	BufferReader             enc.BufferReader
}

type WorkerBee struct {
	link *NDNLPLinkService
	id   int
}

func NewBee(id int, l *NDNLPLinkService) *WorkerBee {
	b := new(WorkerBee)
	b.id = id
	b.link = l
	return b
}
func (b *WorkerBee) Run(jobs <-chan *ndn.PendingPacket) {
	shouldContinue := true
	for shouldContinue {
		select {
		case packet := <-jobs:
			sendPacket(b.link, packet)
		case <-b.link.HasQuit:
			shouldContinue = false
		}
	}
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
	l.pendingAcksToSend = list.New()
	l.idleAckTimer = make(chan interface{}, faceQueueSize)
	l.nextSequence = 0
	l.retransmitQueue = make(chan uint64, faceQueueSize)
	l.rto = 0
	l.nextTxSequence = 0
	l.workQueue = make(chan *ndn.PendingPacket, faceQueueSize)
	for i := 0; i < workers; i++ {
		b := NewBee(i, l)
		go b.Run(l.workQueue)
		l.bees = append(l.bees, b)
	}
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

// Run starts the face and associated goroutines
func (l *NDNLPLinkService) Run(optNewFrame []byte) {
	// Allocate and clear up the memory pool
	pool, err := stealthpool.New(maxPoolBlockCnt, stealthpool.WithBlockSize(maxPoolBlockSize))
	if err != nil {
		core.LogError(l, "Failed to allocate stealthpool")
		return
	}
	defer pool.Close()
	l.stealthPool = pool

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

	// Wait for transport receive goroutine to quit
	//<-l.hasTransportQuit

	l.HasQuit <- true
}
func sendPacket(l *NDNLPLinkService, netPacket *ndn.PendingPacket) {
	wire := netPacket.RawBytes

	if l.transport.State() != ndn.Up {
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
	var fragments []*lpv2.Packet
	if len(wire) > effectiveMtu {
		if !l.options.IsFragmentationEnabled {
			core.LogInfo(l, "Attempted to send frame over MTU on link without fragmentation - DROP")
			return
		}

		// Split up fragment
		nFragments := int(math.Ceil(float64(len(wire)) / float64(effectiveMtu)))
		fragments = make([]*lpv2.Packet, nFragments)
		for i := 0; i < nFragments; i++ {
			if i < nFragments-1 {
				fragments[i] = lpv2.NewPacketNoCopy(wire[effectiveMtu*i : effectiveMtu*(i+1)])
			} else {
				fragments[i] = lpv2.NewPacketNoCopy(wire[effectiveMtu*i:])
			}
		}
	} else {
		fragments = make([]*lpv2.Packet, 1)
		fragments[0] = lpv2.NewPacketNoCopy(wire)
	}

	// Sequence
	if len(fragments) > 1 || l.options.IsReliabilityEnabled {
		for _, fragment := range fragments {
			fragment.SetSequence(l.nextSequence)
			l.nextSequence++
		}
	}

	// Congestion marking
	if congestionMarking && l.transport.GetSendQueueSize() > l.options.DefaultCongestionThresholdBytes && now.After(l.lastTimeCongestionMarked.Add(l.options.BaseCongestionMarkingInterval)) {
		// Mark congestion
		core.LogWarn(l, "Marking congestion")
		fragments[0].SetCongestionMark(1)
		l.lastTimeCongestionMarked = now
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
			unacknowledgedFrame.sentTime = now
			l.unacknowledgedFrames.Store(l.nextTxSequence, unacknowledgedFrame)

			unacknowledgedPacket.unacknowledgedFragments[l.nextTxSequence] = new(interface{})
			l.nextTxSequence++
		}

		unacknowledgedPacket.lock.Unlock()
		l.unacknowledgedPackets.Store(firstSequence, unacknowledgedPacket)
	}

	// PIT tokens
	if len(netPacket.PitToken) > 0 {
		fragments[0].SetPitToken(netPacket.PitToken)
	}

	// Incoming face indication
	if l.options.IsIncomingFaceIndicationEnabled && netPacket.IncomingFaceID != nil {
		fragments[0].SetIncomingFaceID(*netPacket.IncomingFaceID)
	}

	// Congestion marking
	if netPacket.CongestionMark != nil {
		fragments[0].SetCongestionMark(*netPacket.CongestionMark)
	}

	// Fill up remaining space with Acks if Reliability enabled
	/*if l.options.IsReliabilityEnabled {
		// TODO
	}*/

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
}
func (l *NDNLPLinkService) runSend() {
	core.LogTrace(l, "Starting send thread")

	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	if l.options.IsReliabilityEnabled {
		go l.runRetransmit()
		go l.runIdleAckTimer()
	}

	for {
		select {
		case netPacket := <-l.sendQueue:
			go sendPacket(l, netPacket)
			//l.workQueue <- netPacket
		case oldTxSequence := <-l.retransmitQueue:
			loadedFrame, ok := l.unacknowledgedFrames.Load(oldTxSequence)
			if !ok {
				// Frame must have been acknowledged between when noted as timed out and when processed here, so just silently ignore
				continue
			}
			frame := loadedFrame.(*ndnlpUnacknowledgedFrame)
			core.LogDebug(l, "Retransmitting TxSequence=", oldTxSequence, " of Sequence=", frame.netPacket)
			// TODO
		case <-l.idleAckTimer:
			//core.LogTrace(l, "Idle Ack timer expired")
			if l.pendingAcksToSend.Len() > 0 {
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
			}
		case <-l.hasTransportQuit:
			l.hasImplQuit <- true
			return
		}
	}
}

func (l *NDNLPLinkService) handleIncomingFrame(rawFrame []byte) {
	// We have to copy so receive transport buffer can be reused
	wire := make([]byte, len(rawFrame), len(rawFrame))
	copy(wire, rawFrame)
	go l.processIncomingFrame(wire)
}

func (l *NDNLPLinkService) processIncomingFrame(wire []byte) {
	// all incoming frames come through a link service
	// Free up memory
	// Attempt to decode buffer into TLV block
	netPacket := new(ndn.PendingPacket)
	netPacket.IncomingFaceID = new(uint64)
	*netPacket.IncomingFaceID = l.faceID
	packet, _, e := spec.ReadPacket(enc.NewBufferReader(wire))
	if e != nil {
		fmt.Println(e)
		return
	}
	if packet.LpPacket == nil {
		netPacket.RawBytes = wire
		netPacket.EncPacket = packet
	} else {
		fragment := packet.LpPacket.Fragment.Join()
		netPacket.CongestionMark = packet.LpPacket.CongestionMark

		// Consumer-controlled forwarding (NextHopFaceId)
		if l.options.IsConsumerControlledForwardingEnabled && packet.LpPacket.NextHopFaceId != nil {
			netPacket.NextHopFaceID = packet.LpPacket.NextHopFaceId
		}

		// Local cache policy
		if l.options.IsLocalCachePolicyEnabled && &packet.LpPacket.CachePolicy.CachePolicyType != nil {
			netPacket.CachePolicy = &packet.LpPacket.CachePolicy.CachePolicyType
		}

		// PIT Token
		if len(packet.LpPacket.PitToken) > 0 {
			netPacket.PitToken = make([]byte, len(packet.LpPacket.PitToken))
			copy(netPacket.PitToken, packet.LpPacket.PitToken)
		}
		packet, _, e = spec.ReadPacket(enc.NewBufferReader(fragment))
		if e != nil {
			return
		}
		netPacket.RawBytes = fragment
		netPacket.EncPacket = packet
	}
	l.dispatchIncomingPacket(netPacket)

}

func (l *NDNLPLinkService) reassemblePacket(frame *lpv2.Packet, baseSequence uint64, fragIndex uint64, fragCount uint64) []byte {
	_, hasSequence := l.partialMessageStore[baseSequence]
	if !hasSequence {
		// Create map entry
		l.partialMessageStore[baseSequence] = make([][]byte, fragCount)
	}

	// Insert into PartialMessageStore
	l.partialMessageStore[baseSequence][fragIndex] = make([]byte, len(frame.Fragment()))
	copy(l.partialMessageStore[baseSequence][fragIndex], frame.Fragment())

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
		reassembled := make([]byte, receivedTotalLen)
		reassembledSize := 0
		for _, fragment := range l.partialMessageStore[baseSequence] {
			copy(reassembled[reassembledSize:], fragment)
			reassembledSize += len(fragment)
		}

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
					core.LogDebug(l, "Network packet with Sequence number ", frame.netPacket, " exceeded allowed number of retransmissions - DROP")
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

func (op *NDNLPLinkServiceOptions) Flags() (ret uint64) {
	if op.IsConsumerControlledForwardingEnabled {
		ret |= FaceFlagLocalFields
	}
	if op.IsReliabilityEnabled {
		ret |= FaceFlagLpReliabilityEnabled
	}
	if op.IsCongestionMarkingEnabled {
		ret |= FaceFlagCongestionMarking
	}
	return
}
