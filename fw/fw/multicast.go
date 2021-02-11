/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"reflect"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/table"
)

// Multicast is a forwarding strategy that forwards Interests to all nexthop faces.
type Multicast struct {
	StrategyBase
}

func init() {
	strategyTypes = append(strategyTypes, reflect.TypeOf(new(Multicast)))
}

// Instantiate creates a new instance of the Multicast strategy.
func (s *Multicast) Instantiate(fwThread *Thread) {
	name, _ := ndn.NameFromString(StrategyPrefix + "/multicast/%FD%01")
	s.NewStrategyBase(fwThread, name)
}

func (s *Multicast) String() string {
	return "Strategy-Multicast"
}

// GetName ...
func (s *Multicast) GetName() *ndn.Name {
	return s.name
}

// AfterContentStoreHit ...
func (s *Multicast) AfterContentStoreHit(pitEntry *table.PitEntry, inFace uint64, data *ndn.Data) {
	// Send downstream
	core.LogTrace(s, "Forwarding content store hit Data "+data.Name().String()+" to FaceID="+strconv.FormatUint(inFace, 10))
	s.SendData(data, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

// AfterReceiveData ...
func (s *Multicast) AfterReceiveData(pitEntry *table.PitEntry, inFace uint64, data *ndn.Data) {
	for faceID := range pitEntry.InRecords {
		core.LogTrace(s, "Forwarding Data "+data.Name().String()+" to FaceID="+strconv.FormatUint(faceID, 10))
		s.SendData(data, pitEntry, faceID, inFace)
	}
}

// AfterReceiveInterest ...
func (s *Multicast) AfterReceiveInterest(pitEntry *table.PitEntry, inFace uint64, interest *ndn.Interest) {
	nexthops := table.FibStrategyTable.LongestPrefixNexthops(interest.Name())
	if len(nexthops) == 0 {
		core.LogDebug(s, "No nexthop for Interest "+interest.Name().String()+" - DROP")
		return
	}

	for _, nexthop := range nexthops {
		core.LogTrace(s, "Forwarding Interest "+interest.Name().String()+" to FaceID="+strconv.FormatUint(nexthop.Nexthop, 10))
		s.SendInterest(interest, pitEntry, nexthop.Nexthop, inFace)
	}
}

// BeforeSatisfyInterest ...
func (s *Multicast) BeforeSatisfyInterest(pitEntry *table.PitEntry, inFace uint64, data *ndn.Data) {
	// Does nothing in Multicast
	core.LogTrace(s, "BeforeSatisfyInterest: "+data.Name().String()+", FaceID="+strconv.FormatUint(inFace, 10))
}
