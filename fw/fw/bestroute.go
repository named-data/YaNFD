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
	StrategyVersions["best-route"] = []uint64{1}
}

// Instantiate creates a new instance of the BestRoute strategy.
func (s *BestRoute) Instantiate(fwThread *Thread) {
	s.NewStrategyBase(fwThread, ndn.NewGenericNameComponent([]byte("best-route")), 1, "BestRoute")
}

// AfterContentStoreHit ...
func (s *BestRoute) AfterContentStoreHit(pitEntry *table.PitEntry, inFace uint64, data *ndn.Data) {
	// Send downstream
	core.LogTrace(s, "AfterContentStoreHit: Forwarding content store hit Data="+data.Name().String()+" to FaceID="+strconv.FormatUint(inFace, 10))
	s.SendData(data, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

// AfterReceiveData ...
func (s *BestRoute) AfterReceiveData(pitEntry *table.PitEntry, inFace uint64, data *ndn.Data) {
	core.LogTrace(s, "AfterReceiveData: Data="+data.Name().String()+", "+strconv.Itoa(len(pitEntry.InRecords))+" In-Records")
	for faceID := range pitEntry.InRecords {
		core.LogTrace(s, "AfterReceiveData: Forwarding Data="+data.Name().String()+" to FaceID="+strconv.FormatUint(faceID, 10))
		s.SendData(data, pitEntry, faceID, inFace)
	}
}

// AfterReceiveInterest ...
func (s *BestRoute) AfterReceiveInterest(pitEntry *table.PitEntry, inFace uint64, interest *ndn.Interest, nexthops []*table.FibNextHopEntry) {
	if len(nexthops) == 0 {
		core.LogDebug(s, "AfterReceiveInterest: No nexthop for Interest="+interest.Name().String()+" - DROP")
		return
	}

	lowestCost := nexthops[0]
	for _, nexthop := range nexthops {
		if nexthop.Cost < lowestCost.Cost {
			lowestCost = nexthop
		}
	}

	core.LogTrace(s, "AfterReceiveInterest: Forwarding Interest="+interest.Name().String()+" to FaceID="+strconv.FormatUint(lowestCost.Nexthop, 10))
	s.SendInterest(interest, pitEntry, lowestCost.Nexthop, inFace)
}
