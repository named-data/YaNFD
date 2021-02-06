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

// BestRoute is a forwarding strategy that forwards Interests to the nexthop with the lowest cost.
type BestRoute struct {
	StrategyBase
}

func init() {
	strategyTypes = append(strategyTypes, reflect.TypeOf(new(BestRoute)))
}

// Instantiate creates a new instance of the BestRoute strategy.
func (s *BestRoute) Instantiate(fwThread *Thread) {
	name, _ := ndn.NameFromString(StrategyPrefix + "/best-route/%FD%01")
	s.NewStrategyBase(fwThread, name)
}

func (s *BestRoute) String() string {
	return "Strategy-BestRoute"
}

// GetName ...
func (s *BestRoute) GetName() *ndn.Name {
	return s.name
}

// AfterContentStoreHit ...
func (s *BestRoute) AfterContentStoreHit(pitEntry *table.PitEntry, inFace int, data *ndn.Data) {
	// Send downstream
	core.LogTrace(s, "AfterContentStoreHit: Forwarding content store hit Data "+data.Name().String()+" to "+strconv.Itoa(inFace))
	s.SendData(data, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

// AfterReceiveData ...
func (s *BestRoute) AfterReceiveData(pitEntry *table.PitEntry, inFace int, data *ndn.Data) {
	core.LogTrace(s, "AfterReceiveData: "+data.Name().String()+", "+strconv.Itoa(len(pitEntry.InRecords))+" In-Records")
	for faceID := range pitEntry.InRecords {
		core.LogTrace(s, "Forwarding Data "+data.Name().String()+" to "+strconv.Itoa(faceID))
		s.SendData(data, pitEntry, faceID, inFace)
	}
}

// AfterReceiveInterest ...
func (s *BestRoute) AfterReceiveInterest(pitEntry *table.PitEntry, inFace int, interest *ndn.Interest) {
	nexthops := table.FibStrategyTable.LongestPrefixNexthops(interest.Name())
	if len(nexthops) == 0 {
		core.LogDebug(s, "No nexthop for Interest "+interest.Name().String()+" - DROP")
		return
	}

	lowestCost := nexthops[0]
	for _, nexthop := range nexthops {
		if nexthop.Cost < lowestCost.Cost {
			lowestCost = nexthop
		}
	}

	core.LogTrace(s, "AfterReceiveInterest: Forwarding Interest "+interest.Name().String()+" to "+strconv.Itoa(lowestCost.Nexthop))
	s.SendInterest(interest, pitEntry, lowestCost.Nexthop, inFace)
}

// BeforeSatisfyInterest ...
func (s *BestRoute) BeforeSatisfyInterest(pitEntry *table.PitEntry, inFace int, data *ndn.Data) {
	// Does nothing in BestRoute
	core.LogTrace(s, "BeforeSatisfyInterest: "+data.Name().String())
}
