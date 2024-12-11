/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"reflect"
	"sort"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/defn"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// BestRoute is a forwarding strategy that forwards Interests
// to the nexthop with the lowest cost.
type BestRoute struct {
	StrategyBase
}

func init() {
	strategyTypes = append(strategyTypes, reflect.TypeOf(new(BestRoute)))
	StrategyVersions["best-route"] = []uint64{1}
}

func (s *BestRoute) Instantiate(fwThread *Thread) {
	s.NewStrategyBase(fwThread, enc.Component{
		Typ: enc.TypeGenericNameComponent, Val: []byte("best-route"),
	}, 1, "BestRoute")
}

func (s *BestRoute) AfterContentStoreHit(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
) {
	core.LogTrace(s, "AfterContentStoreHit: Forwarding content store hit Data=", packet.Name, " to FaceID=", inFace)
	s.SendData(packet, pitEntry, inFace, 0) // 0 indicates ContentStore is source
}

func (s *BestRoute) AfterReceiveData(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
) {
	core.LogTrace(s, "AfterReceiveData: Data=", ", ", len(pitEntry.InRecords()), " In-Records")
	for faceID := range pitEntry.InRecords() {
		core.LogTrace(s, "AfterReceiveData: Forwarding Data=", packet.Name, " to FaceID=", faceID)
		s.SendData(packet, pitEntry, faceID, inFace)
	}
}

func (s *BestRoute) AfterReceiveInterest(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	inFace uint64,
	nexthops []*table.FibNextHopEntry,
) {
	sort.Slice(nexthops, func(i, j int) bool { return nexthops[i].Cost < nexthops[j].Cost })
	for _, nh := range nexthops {
		core.LogTrace(s, "AfterReceiveInterest: Forwarding Interest=", packet.Name, " to FaceID=", nh.Nexthop)
		if sent := s.SendInterest(packet, pitEntry, nh.Nexthop, inFace); sent {
			return
		}
	}

	core.LogDebug(s, "AfterReceiveInterest: No usable nexthop for Interest=", packet.Name, " - DROP")
}

func (s *BestRoute) BeforeSatisfyInterest(pitEntry table.PitEntry, inFace uint64) {
	// This does nothing in BestRoute
}
