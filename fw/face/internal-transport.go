/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"runtime"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/lpv2"
	"github.com/eric135/YaNFD/ndn/tlv"
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
	t.makeTransportBase(ndn.MakeInternalFaceURI(), ndn.MakeInternalFaceURI(), PersistencyPersistent, ndn.Local, ndn.PointToPoint, tlv.MaxNDNPacketSize)
	t.recvQueue = make(chan []byte, faceQueueSize)
	t.sendQueue = make(chan []byte, faceQueueSize)
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
	return "InternalTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
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
func (t *InternalTransport) Send(block *tlv.Block, pitToken []byte, nextHopFaceID *uint64) {
	netWire, err := block.Wire()
	if err != nil {
		core.LogWarn(t, "Unable to decode net packet to send - DROP")
		return
	}
	lpPacket := lpv2.NewPacket(netWire)
	if len(pitToken) > 0 {
		lpPacket.SetPitToken(pitToken)
	}
	if nextHopFaceID != nil {
		lpPacket.SetNextHopFaceID(*nextHopFaceID)
	}
	lpPacketWire, err := lpPacket.Encode()
	if err != nil {
		core.LogWarn(t, "Unable to encode block to send - DROP")
		return
	}
	frame, err := lpPacketWire.Wire()
	if err != nil {
		core.LogWarn(t, "Unable to encode block to send - DROP")
		return
	}
	t.sendQueue <- frame
}

// Receive receives a packet from the perspective of the internal component.
func (t *InternalTransport) Receive() (*tlv.Block, []byte, uint64) {
	shouldContinue := true
	// We need to use a for loop to silently ignore invalid packets
	for shouldContinue {
		select {
		case frame := <-t.recvQueue:
			lpBlock, _, err := tlv.DecodeBlock(frame)
			if err != nil {
				core.LogWarn(t, "Unable to decode received block - DROP")
				continue
			}
			lpPacket, err := lpv2.DecodePacket(lpBlock)
			if err != nil {
				core.LogWarn(t, "Unable to decode received block - DROP")
				continue
			}
			if len(lpPacket.Fragment()) == 0 {
				core.LogWarn(t, "Received empty fragment - DROP")
				continue
			}

			block, _, err := tlv.DecodeBlock(lpPacket.Fragment())
			if err != nil {
				core.LogWarn(t, "Unable to decode received block - DROP")
				continue
			}
			return block, lpPacket.PitToken(), *lpPacket.IncomingFaceID()
		case <-t.hasQuit:
			shouldContinue = false
		}
	}
	return nil, []byte{}, 0
}

func (t *InternalTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	t.nOutBytes += uint64(len(frame))

	core.LogDebug(t, "Sending frame of size ", len(frame))
	t.recvQueue <- frame
}

func (t *InternalTransport) runReceive() {
	core.LogTrace(t, "Starting receive thread")

	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	for {
		core.LogTrace(t, "Waiting for frame from component")
		select {
		case <-t.hasQuit:
			return
		case frame := <-t.sendQueue:
			core.LogTrace(t, "Component send of size ", len(frame))

			if len(frame) > tlv.MaxNDNPacketSize {
				core.LogWarn(t, "Component trying to send too much data - DROP")
				continue
			}

			t.nInBytes += uint64(len(frame))

			t.linkService.handleIncomingFrame(frame)
		}
	}
}

func (t *InternalTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn.Up {
		// Stop link service
		t.hasQuit <- true
		t.hasQuit <- true // Send again to stop any pending receives
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
