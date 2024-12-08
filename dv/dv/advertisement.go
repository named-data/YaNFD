package dv

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema/svs"
	"github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

func (dv *DV) Advertise() (err error) {
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
		HopLimit:    utils.IdPtr(uint(1)),
	}

	// State Vector for our group
	sv := &svs.StateVec{
		Entries: []*svs.StateVecEntry{{
			NodeId: dv.routerPrefix.Bytes(),
			SeqNo:  dv.advSeq,
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

func (dv *DV) onAdvSyncInterest(
	interest ndn.Interest,
	rawInterest enc.Wire,
	sigCovered enc.Wire,
	reply ndn.ReplyFunc,
	deadline time.Time,
) {
	// TODO: verify signature on Sync Interest

	// Parse the state vector from app parameters
	if interest.AppParam() == nil {
		log.Warn("onAdvSyncInterest: Received Sync Interest with no AppParam, ignoring")
		return
	}

	sv, err := svs.ParseStateVec(enc.NewWireReader(interest.AppParam()), true)
	if err != nil {
		log.Warnf("onAdvSyncInterest: Failed to parse StateVec: %+v", err)
		return
	}

	// Process each entry in the state vector
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	for _, entry := range sv.Entries {
		// Parse name from NodeId
		nodeId, err := enc.NameFromBytes(entry.NodeId)
		if err != nil {
			log.Warnf("onAdvSyncInterest: Failed to parse NodeId: %+v", err)
			continue
		}

		// Check if the entry is newer than what we know
		hash := nodeId.Hash()
		if known, ok := dv.neighborAdvSeq[hash]; ok {
			if known >= entry.SeqNo {
				// Nothing has changed, skip
				continue
			}
		}

		dv.neighborAdvSeq[hash] = entry.SeqNo
		go dv.scheduleAdvFetch(nodeId, entry.SeqNo)
	}
}

func (dv *DV) scheduleAdvFetch(nodeId enc.Name, seqNo uint64) {
	// debounce; wait before fetching, then check if this is still the latest
	// sequence number known for this neighbor
	time.Sleep(10 * time.Millisecond)
	if known, ok := dv.neighborAdvSeq[nodeId.Hash()]; !ok || known != seqNo {
		return
	}

	advName := append(nodeId,
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
		log.Warnf("scheduleAdvFetch: Failed to make Interest: %+v", err)
		return
	}

	// Fetch the advertisement
	err = dv.engine.Express(finalName, cfg, wire, func(result ndn.InterestResult, data ndn.Data, _ enc.Wire, _ enc.Wire, _ uint64) {
		if result != ndn.InterestResultData {
			// Keep retrying until we get the advertisement
			// If the router is dead, we break out of this by checking
			// that the sequence number is gone (above)
			log.Warnf("scheduleAdvFetch: Retrying: %+v", result)
			go dv.scheduleAdvFetch(nodeId, seqNo)
			return
		}

		// Process the advertisement
		go dv.onAdvData(data)
	})
	if err != nil {
		log.Warnf("scheduleAdvFetch: Failed to express Interest: %+v", err)
	}
}

// Received advertisement Interest
func (dv *DV) onAdvInterest(
	interest ndn.Interest,
	rawInterest enc.Wire,
	sigCovered enc.Wire,
	reply ndn.ReplyFunc,
	deadline time.Time,
) {
	// For now, just send the latest advertisement at all times
	// This will need to change if we switch to differential updates

	// TODO: sign the advertisement
	signer := security.NewSha256Signer()

	// TODO: encode the advertisement
	content := []byte("Hello, world!")

	// Make the Data packet
	wire, _, err := dv.engine.Spec().MakeData(
		interest.Name(),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(10 * time.Second),
		},
		enc.Wire{content},
		signer)
	if err != nil {
		log.Warnf("onAdvInterest: Failed to make Data: %+v", err)
		return
	}

	// Send the Data packet
	err = reply(wire)
	if err != nil {
		log.Warnf("onAdvInterest: Failed to reply: %+v", err)
		return
	}
}

// Received advertisement Data
func (dv *DV) onAdvData(data ndn.Data) {
	name := data.Name()
	neighbor := name[:len(name)-3]
	seqNo := name[len(name)-1].NumberVal()

	// Check if this is the latest advertisement
	if known, ok := dv.neighborAdvSeq[neighbor.Hash()]; !ok || known != seqNo {
		// This is an old advertisement, sad
		log.Warnf("onAdvData: Received old advertisement for %s, seqNo %d", neighbor.String(), seqNo)
		return
	}

	// TODO: verify signature on Advertisement

	log.Infof("onAdvData: Received Advertisement: %s", data.Name().String())
}
