/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"strconv"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/table"
)

// StrategyPrefix is the prefix of all strategy names for YaNFD
const StrategyPrefix = "/localhost/yanfd/strategy"

// Strategy represents a forwarding strategy.
type Strategy interface {
	Instantiate(fwThread *Thread)
	GetName() *ndn.Name

	AfterContentStoreHit(pitEntry *table.PitEntry, inFace int, data *ndn.Data)
	AfterReceiveData(pitEntry *table.PitEntry, inFace int, data *ndn.Data)
	AfterReceiveInterest(pitEntry *table.PitEntry, inFace int, interest *ndn.Interest)
	BeforeSatisfyInterest(pitEntry *table.PitEntry, inFace int, interest *ndn.Data)
}

// StrategyBase provides common helper methods for YaNFD forwarding strategies.
type StrategyBase struct {
	thread   *Thread
	threadID int
	name     *ndn.Name
}

// NewStrategyBase is a helper that allows specific strategies to initialize the base.
func (s *StrategyBase) NewStrategyBase(fwThread *Thread, name *ndn.Name) {
	s.thread = fwThread
	s.threadID = s.thread.threadID
	s.name = name
}

func (s *StrategyBase) String() string {
	return "StrategyBase-" + strconv.Itoa(s.threadID)
}

// SendInterest sends an Interest on the specified face.
func (s *StrategyBase) SendInterest(interest *ndn.Interest, pitEntry *table.PitEntry, nexthop int, inFace int) {
	s.thread.processOutgoingInterest(interest, pitEntry, nexthop, inFace)
}

// SendData sends a Data packet on the specified face.
func (s *StrategyBase) SendData(data *ndn.Data, pitEntry *table.PitEntry, nexthop int, inFace int) {
	var pitToken []byte
	if inRecord, ok := pitEntry.InRecords[nexthop]; ok {
		pitToken = inRecord.PitToken
		delete(pitEntry.InRecords, nexthop)
	}
	s.thread.processOutgoingData(data, nexthop, pitToken, inFace)
}
