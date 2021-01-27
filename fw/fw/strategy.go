/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"encoding/binary"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/dispatch"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/table"
)

// StrategyPrefix is the prefix of all strategy names for YaNFD
const StrategyPrefix = "/localhost/yanfd/strategy"

// Strategy represents a forwarding strategy.
type Strategy interface {
	GetName() *ndn.Name

	AfterContentStoreHit(pitEntry *table.PitEntry, inFace int, data *ndn.Data)
	AfterReceiveData(pitEntry *table.PitEntry, inFace int, data *ndn.Data)
	AfterReceiveInterest(pitEntry *table.PitEntry, inFace int, interest *ndn.Interest)
	BeforeSatisfyInterest(pitEntry *table.PitEntry, inFace int, interest *ndn.Data)
}

// StrategyBase provides common helper methods for YaNFD forwarding strategies.
type StrategyBase struct {
	ThreadID int
}

// NewStrategyBase is a helper that allows specific strategies to initialize the base.
func (s *StrategyBase) NewStrategyBase(threadID int) {
	s.ThreadID = threadID
}

func (s *StrategyBase) String() string {
	return "StrategyBase-" + strconv.Itoa(s.ThreadID)
}

// SendInterest sends an Interest on the specified face.
func (s *StrategyBase) SendInterest(interest *ndn.Interest, faceID int) {
	pendingPacket := new(ndn.PendingPacket)
	pendingPacket.PitToken = make([]byte, 2)
	binary.BigEndian.PutUint16(pendingPacket.PitToken, uint16(s.ThreadID))
	var err error
	pendingPacket.Wire, err = interest.Encode()
	if err != nil {
		core.LogWarn(s, "Unable to encode Interest "+interest.Name().String()+" before sending - DROP")
	}
	dispatch.GetFace(faceID).SendPacket(pendingPacket)
}

// SendData sends a Data packet on the specified face.
func (s *StrategyBase) SendData(data *ndn.Data, pitEntry *table.PitEntry, faceID int) {
	pendingPacket := new(ndn.PendingPacket)
	if inRecord, ok := pitEntry.InRecords[faceID]; ok {
		pendingPacket.PitToken = inRecord.PitToken
	}
	var err error
	pendingPacket.Wire, err = data.Encode()
	if err != nil {
		core.LogWarn(s, "Unable to encode Data "+data.Name().String()+" before sending - DROP")
	}
	dispatch.GetFace(faceID).SendPacket(pendingPacket)
}
