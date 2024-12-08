package dv

import (
	"time"

	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema/svs"
	"github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type neighbor_state struct {
	// neighbor name
	name enc.Name
	// time of last sync interest
	lastSeen time.Time
	// advertisement sequence number for neighbor
	advertSeq uint64
	// most recent advertisement
	advert *tlv.Advertisement
	// most recent advertisement (wire)
	advertWire []byte
}

func (dv *DV) sendAdvertSyncInterest() (err error) {
	// SVS v2 Sync Interest
	syncName := append(dv.globalPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
		enc.NewVersionComponent(2),
	)

	// Sync Interest parameters for SVS
	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    utils.IdPtr(1 * time.Millisecond),
		Nonce:       utils.ConvertNonce(dv.engine.Timer().Nonce()),
		HopLimit:    utils.IdPtr(uint(1)), // TODO: use localhop instead
	}

	// State Vector for our group
	sv := &svs.StateVec{
		Entries: []*svs.StateVecEntry{{
			NodeId: dv.routerPrefix.Bytes(),
			SeqNo:  dv.advertSeq,
		}},
	}

	// TODO: sign the sync interest

	wire, _, finalName, err := dv.engine.Spec().MakeInterest(syncName, cfg, sv.Encode(), nil)
	if err != nil {
		return err
	}

	// Sync Interest has no reply
	err = dv.engine.Express(finalName, cfg, wire, nil)
	if err != nil {
		return err
	}

	return nil
}

func (dv *DV) onAdvertSyncInterest(
	interest ndn.Interest,
	reply ndn.ReplyFunc,
	extra ndn.InterestHandlerExtra,
) {
	// Check if app param is present
	if interest.AppParam() == nil {
		log.Warn("onAdvertSyncInterest: Received Sync Interest with no AppParam, ignoring")
		return
	}

	// If there is no incoming face ID, we can't use this
	if extra.IncomingFaceId == nil {
		log.Warn("onAdvertSyncInterest: Received Sync Interest with no incoming face ID, ignoring")
		return
	}

	// TODO: verify signature on Sync Interest

	sv, err := svs.ParseStateVec(enc.NewWireReader(interest.AppParam()), true)
	if err != nil {
		log.Warnf("onAdvertSyncInterest: Failed to parse StateVec: %+v", err)
		return
	}

	// Process each entry in the state vector
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	for _, entry := range sv.Entries {
		// Parse name from NodeId
		neighbor, err := enc.NameFromBytes(entry.NodeId)
		if err != nil {
			log.Warnf("onAdvertSyncInterest: Failed to parse NodeId: %+v", err)
			continue
		}

		// Check if the entry is newer than what we know
		nhash := neighbor.Hash()
		ns := dv.neighbors[nhash]
		if ns != nil {
			if ns.advertSeq >= entry.SeqNo {
				// Nothing has changed, skip
				ns.lastSeen = time.Now()
				continue
			}
		} else {
			// Create new neighbor entry cause none found
			// This is the ONLY place where neighbors are created
			// In all other places, quit if not found
			ns = &neighbor_state{
				name: neighbor.Clone(),
			}
			dv.neighbors[nhash] = ns
		}

		ns.advertSeq = entry.SeqNo
		ns.lastSeen = time.Now()

		go dv.scheduleAdvertFetch(neighbor, entry.SeqNo)
	}
}

func (dv *DV) scheduleAdvertFetch(neighbor enc.Name, seqNo uint64) {
	// debounce; wait before fetching, then check if this is still the latest
	// sequence number known for this neighbor
	time.Sleep(10 * time.Millisecond)
	if ns, ok := dv.neighbors[neighbor.Hash()]; !ok || ns.advertSeq != seqNo {
		return
	}

	advName := append(neighbor,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
		enc.NewSequenceNumComponent(seqNo), // unused for now
	)

	// Fetch the advertisement
	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    utils.IdPtr(4 * time.Second),
		Nonce:       utils.ConvertNonce(dv.engine.Timer().Nonce()),
	}

	wire, _, finalName, err := dv.engine.Spec().MakeInterest(advName, cfg, nil, nil)
	if err != nil {
		log.Warnf("scheduleAdvertFetch: Failed to make Interest: %+v", err)
		return
	}

	// Fetch the advertisement
	err = dv.engine.Express(finalName, cfg, wire, func(result ndn.InterestResult, data ndn.Data, _ enc.Wire, _ enc.Wire, _ uint64) {
		if result != ndn.InterestResultData {
			// Keep retrying until we get the advertisement
			// If the router is dead, we break out of this by checking
			// that the sequence number is gone (above)
			log.Warnf("scheduleAdvertFetch: Retrying: %+v", result)
			go dv.scheduleAdvertFetch(neighbor, seqNo)
			return
		}

		// Process the advertisement
		go dv.onAdvertData(data)
	})
	if err != nil {
		log.Warnf("scheduleAdvertFetch: Failed to express Interest: %+v", err)
	}
}

// Received advertisement Interest
func (dv *DV) onAdvertInterest(
	interest ndn.Interest,
	reply ndn.ReplyFunc,
	extra ndn.InterestHandlerExtra,
) {
	// For now, just send the latest advertisement at all times
	// This will need to change if we switch to differential updates

	// TODO: sign the advertisement
	signer := security.NewSha256Signer()

	// Encode latest advertisement
	content := func() *tlv.Advertisement {
		dv.mutex.Lock()
		defer dv.mutex.Unlock()
		return dv.rib.advert()
	}().Encode()

	wire, _, err := dv.engine.Spec().MakeData(
		interest.Name(),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(10 * time.Second),
		},
		content,
		signer)
	if err != nil {
		log.Warnf("onAdvertInterest: Failed to make Data: %+v", err)
		return
	}

	// Send the Data packet
	err = reply(wire)
	if err != nil {
		log.Warnf("onAdvertInterest: Failed to reply: %+v", err)
		return
	}
}

// Received advertisement Data
func (dv *DV) onAdvertData(data ndn.Data) {
	name := data.Name()
	neighbor := name[:len(name)-3]
	seqNo := name[len(name)-1].NumberVal()
	nhash := neighbor.Hash()

	// Check if this is the latest advertisement
	if ns, ok := dv.neighbors[nhash]; !ok || ns.advertSeq != seqNo {
		log.Warnf("onAdvertData: Received old advertisement for %s, seqNo %d", neighbor.String(), seqNo)
		return
	}

	// TODO: verify signature on Advertisement
	log.Infof("onAdvertData: Received Advertisement: %s", data.Name().String())

	// Parse the advertisement
	raw := data.Content().Join() // clone
	advert, err := tlv.ParseAdvertisement(enc.NewWireReader(enc.Wire{raw}), false)
	if err != nil {
		log.Warnf("onAdvertData: Failed to parse Advertisement: %+v", err)
		return
	}

	// Update the local advertisement list
	dv.mutex.Lock()
	defer dv.mutex.Unlock()
	dv.neighbors[nhash].advert = advert
	dv.neighbors[nhash].advertWire = raw
	go dv.UpdateRIB(dv.neighbors[nhash])
}

// Compute the RIB
func (dv *DV) UpdateRIB(ns *neighbor_state) {
	log.Infof("UpdateRIB: Triggered for %s", ns.name.String())

	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	if ns.advert == nil {
		return
	}

	// TODO: use cost to neighbor
	localCost := uint64(1)

	// Trigger our own advertisement if needed
	var dirty bool = false

	// Reset destinations for this neighbor
	dv.rib.dirtyResetNextHop(ns.name)

	for _, entry := range ns.advert.Entries {
		// Use the advertised cost by default
		cost := entry.Cost + localCost

		// Poison reverse - try other cost if next hop is us
		if entry.NextHop.Name.Equal(dv.routerPrefix) {
			if entry.OtherCost < CostInfinity {
				cost = entry.OtherCost + localCost
			} else {
				cost = CostInfinity
			}
		}

		// Skip unreachable destinations
		if cost >= CostInfinity {
			continue
		}

		// Check advertisement changes
		dirty = dv.rib.set(entry.Destination.Name, ns.name, cost) || dirty
	}

	// Drop dead entries
	dirty = dv.rib.prune() || dirty

	// If advert changed, increment sequence number
	if dirty {
		go dv.notifyNewAdvert()
	}
}

// Check for dead neighbors
func (dv *DV) checkDeadNeighbors() {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	dirty := false
	for nhash, ns := range dv.neighbors {
		if time.Since(ns.lastSeen) > dv.config.RouterDeadInterval {
			log.Infof("checkDeadNeighbors: Neighbor %s is dead", ns.name.String())

			// Remove neighbor from RIB and prune
			dirty = dv.rib.removeNextHop(ns.name) || dirty
			dirty = dv.rib.prune() || dirty

			// This is the ONLY place that can remove the neighbor
			// from the state list.
			delete(dv.neighbors, nhash)
		}
	}

	if dirty {
		go dv.notifyNewAdvert()
	}
}

func (dv *DV) notifyNewAdvert() {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	dv.rib.Print()

	dv.advertSeq++
	go dv.sendAdvertSyncInterest()
}
