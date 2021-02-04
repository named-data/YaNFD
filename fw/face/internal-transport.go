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
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/lpv2"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// InternalTransport is a transport for use by internal YaNFD modules (e.g., management).
type InternalTransport struct {
	recvQueue chan []byte             // Contains pending packets sent to internal component
	sendQueue chan *ndn.PendingPacket // Contains pending packets sent by the internal component
	transportBase
}

// MakeInternalTransport makes an InternalTransport.
func MakeInternalTransport() *InternalTransport {
	t := new(InternalTransport)
	t.makeTransportBase(ndn.MakeNullFaceURI(), ndn.MakeNullFaceURI(), tlv.MaxNDNPacketSize)
	t.recvQueue = make(chan []byte)
	t.sendQueue = make(chan *ndn.PendingPacket)
	t.changeState(ndn.Up)
	return t
}

// RegisterInternalTransport creates, registers, and starts an InternalTransport.
func RegisterInternalTransport() (LinkService, *InternalTransport) {
	t := MakeInternalTransport()
	l := MakeNDNLPLinkService(t, NDNLPLinkServiceOptions{
		IsIncomingFaceIndicationEnabled:       true,
		IsConsumerControlledForwardingEnabled: true,
	})
	FaceTable.Add(l)
	go l.Run()
	return l, t
}

func (t *InternalTransport) String() string {
	return "InternalTransport, FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// Send sends a packet from the perspective of the internal component.
func (t *InternalTransport) Send(block *tlv.Block, nextHopFaceID *int) {
	pendingPacket := new(ndn.PendingPacket)
	pendingPacket.Wire = block
	if nextHopFaceID != nil {
		pendingPacket.NextHopFaceID = new(uint64)
		*pendingPacket.NextHopFaceID = uint64(*nextHopFaceID)
	}
	t.sendQueue <- pendingPacket
}

// Receive receives a packet from the perspective of the internal component.
func (t *InternalTransport) Receive() (*tlv.Block, int) {
	shouldContinue := true
	// We need to use a for loop to silently ignore invalid packets
	for shouldContinue {
		select {
		case frame := <-t.recvQueue:
			lpBlock, _, err := tlv.DecodeBlock(frame)
			if err != nil {
				core.LogWarn(t, "Unable to decode block")
				continue
			}
			lpPacket, err := lpv2.DecodePacket(lpBlock)
			if err != nil {
				core.LogWarn(t, "Unable to decode block")
				continue
			}
			if len(lpPacket.Fragment()) == 0 {
				core.LogWarn(t, "Sent empty fragment")
				continue
			}

			block, _, err := tlv.DecodeBlock(lpPacket.Fragment())
			if err != nil {
				core.LogWarn(t, "Unable to decode block")
				continue
			}
			return block, int(*lpPacket.IncomingFaceID())
		case <-t.hasQuit:
			shouldContinue = false
		}
	}
	return nil, 0
}

func (t *InternalTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	t.recvQueue <- frame
}

func (t *InternalTransport) runReceive() {
	core.LogTrace(t, "Starting receive thread")
	for {
		core.LogTrace(t, "Waiting for frame from component")
		select {
		case <-t.hasQuit:
			return
		case pendingPacket := <-t.sendQueue:
			core.LogTrace(t, "Receive of size "+strconv.Itoa(pendingPacket.Wire.Size()))

			if pendingPacket.Wire.Size() > tlv.MaxNDNPacketSize {
				core.LogWarn(t, "Received too much data without valid TLV block - DROP")
				continue
			}

			// Packet was successfully received, send up to link service
			if frame, err := pendingPacket.Wire.Wire(); err != nil {
				core.LogWarn(t, "Unable to encode frame from component ("+err.Error()+" - DROP")
			} else {
				t.linkService.handleIncomingFrame(frame)
			}
		}
	}
}

func (t *InternalTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "- state:", t.state, "->", new)
	t.state = new

	if t.state != ndn.Up {
		// Stop link service
		t.hasQuit <- true
		t.hasQuit <- true // Send again to stop any pending receives
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
