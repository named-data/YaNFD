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
	defn "github.com/named-data/YaNFD/defn"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// InternalTransport is a transport for use by internal YaNFD modules (e.g., management).
type InternalTransport struct {
	recvQueue chan []byte // Contains pending packets sent to internal component
	sendQueue chan []byte // Contains pending packets sent by the internal component
	transportBase
}

// MakeInternalTransport makes an InternalTransport.
func MakeInternalTransport() *InternalTransport {
	t := new(InternalTransport)
	t.makeTransportBase(
		defn.MakeInternalFaceURI(),
		defn.MakeInternalFaceURI(),
		PersistencyPersistent,
		defn.Local,
		defn.PointToPoint,
		defn.MaxNDNPacketSize)
	t.recvQueue = make(chan []byte, faceQueueSize)
	t.sendQueue = make(chan []byte, faceQueueSize)
	t.running.Store(true)
	return t
}

// RegisterInternalTransport creates, registers, and starts an InternalTransport.
func RegisterInternalTransport() (LinkService, *InternalTransport) {
	transport := MakeInternalTransport()

	options := MakeNDNLPLinkServiceOptions()
	options.IsIncomingFaceIndicationEnabled = true
	options.IsConsumerControlledForwardingEnabled = true
	link := MakeNDNLPLinkService(transport, options)
	link.Run(nil)

	return link, transport
}

func (t *InternalTransport) String() string {
	return "InternalTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) +
		", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *InternalTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	if persistency == PersistencyPersistent {
		t.persistency = persistency
		return true
	}

	return false
}

// GetSendQueueSize returns the current size of the send queue.
func (t *InternalTransport) GetSendQueueSize() uint64 {
	return 0
}

// Send sends a packet from the perspective of the internal component.
func (t *InternalTransport) Send(netWire enc.Wire, pitToken []byte, nextHopFaceID *uint64) {
	lpPkt := &spec.LpPacket{
		Fragment: netWire,
	}
	if len(pitToken) > 0 {
		lpPkt.PitToken = pitToken
	}
	if nextHopFaceID != nil {
		lpPkt.NextHopFaceId = utils.IdPtr(*nextHopFaceID)
	}
	pkt := &spec.Packet{
		LpPacket: lpPkt,
	}
	encoder := spec.PacketEncoder{}
	encoder.Init(pkt)
	lpPacketWire := encoder.Encode(pkt)
	if lpPacketWire == nil {
		core.LogWarn(t, "Unable to encode block to send - DROP")
		return
	}
	t.sendQueue <- lpPacketWire.Join()
}

// Receive receives a packet from the perspective of the internal component.
func (t *InternalTransport) Receive() (enc.Wire, []byte, uint64) {
	for frame := range t.recvQueue {
		packet, _, err := spec.ReadPacket(enc.NewBufferReader(frame))
		if err != nil {
			core.LogWarn(t, "Unable to decode received block - DROP: ", err)
			continue
		}

		lpPkt := packet.LpPacket
		if lpPkt.Fragment.Length() == 0 {
			core.LogWarn(t, "Received empty fragment - DROP")
			continue
		}

		return lpPkt.Fragment, lpPkt.PitToken, *lpPkt.IncomingFaceId
	}

	return nil, []byte{}, 0
}

func (t *InternalTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	t.nOutBytes += uint64(len(frame))

	frameCopy := make([]byte, len(frame))
	copy(frameCopy, frame)
	t.recvQueue <- frameCopy
}

func (t *InternalTransport) runReceive() {
	for frame := range t.sendQueue {
		if len(frame) > defn.MaxNDNPacketSize {
			core.LogWarn(t, "Component trying to send too much data - DROP")
			continue
		}

		t.nInBytes += uint64(len(frame))
		t.linkService.handleIncomingFrame(frame)
	}
}

func (t *InternalTransport) Close() {
	if t.running.Swap(false) {
		close(t.recvQueue)
		close(t.sendQueue)
	}
}
