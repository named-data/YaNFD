package dv

import (
	"time"

	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	ndn_sync "github.com/zjkmxy/go-ndn/pkg/engine/sync"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// Fetch all required prefix data
func (dv *Router) prefixDataFetchAll() {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	for _, e := range dv.rib.Entries() {
		router := dv.pfx.GetRouter(e.Name())
		if router.Known < router.Latest {
			go dv.prefixDataFetch(e.Name())
		}
	}
}

// Received prefix sync update
func (dv *Router) onPfxSyncUpdate(ssu ndn_sync.SvSyncUpdate) {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Update the prefix table
	dv.pfx.GetRouter(ssu.NodeId).Latest = ssu.High

	// Start a fetching thread (if needed)
	go dv.prefixDataFetch(ssu.NodeId)
}

// Fetch prefix data
func (dv *Router) prefixDataFetch(nodeId enc.Name) {
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Check if the RIB has this destination
	if !dv.rib.Has(nodeId) {
		return
	}

	// At any given time, there is only one thread fetching
	// prefix data for a node. This thread recursively calls itself.
	router := dv.pfx.GetRouter(nodeId)
	if router.Fetching || router.Known >= router.Latest {
		return
	}

	// Mark this node as fetching
	router.Fetching = true

	// Fetch the prefix data
	log.Debugf("Fetching prefix data for %s [%d => %d]", nodeId, router.Known, router.Latest)

	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    utils.IdPtr(2 * time.Second),
		Nonce:       utils.ConvertNonce(dv.engine.Timer().Nonce()),
	}

	isSnap := router.Latest-router.Known > 100
	name := append(nodeId,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "PFX"),
	)
	if isSnap {
		name = append(name, enc.NewStringComponent(enc.TypeKeywordNameComponent, "SNAP"))
		cfg.CanBePrefix = true
	} else {
		name = append(name, enc.NewSequenceNumComponent(router.Known+1))
	}

	wire, _, finalName, err := dv.engine.Spec().MakeInterest(name, cfg, nil, nil)
	if err != nil {
		log.Warnf("prefixDataFetch: Failed to make Interest: %+v", err)
		return
	}

	err = dv.engine.Express(finalName, cfg, wire, func(
		result ndn.InterestResult,
		data ndn.Data,
		_, _ enc.Wire, _ uint64,
	) {
		go func() {
			dv.mutex.Lock()
			defer dv.mutex.Unlock()

			// Done fetching, restart if needed
			defer func() {
				router.Fetching = false
				go dv.prefixDataFetch(nodeId) // recheck
			}()

			if result != ndn.InterestResultData {
				log.Warnf("prefixDataFetch: Failed to fetch prefix data %s: %d", finalName, result)

				// see advertDataFetch
				if result != ndn.InterestResultTimeout {
					time.Sleep(2 * time.Second)
				} else {
					time.Sleep(100 * time.Millisecond)
				}
				return
			}

			ops, err := tlv.ParsePrefixOpList(enc.NewWireReader(data.Content()), true)
			if err != nil {
				log.Warnf("prefixDataFetch: Failed to parse PrefixOpList: %+v", err)
				return
			}

			// Update sequence number
			dataName := data.Name()
			seqNo := dataName[len(dataName)-1]
			if seqNo.Typ != enc.TypeSequenceNumNameComponent {
				log.Warnf("prefixDataFetch: Unexpected sequence number type: %s", seqNo.Typ)
				return
			}

			dv.pfx.Apply(ops)
			router.Known = seqNo.NumberVal()
		}()
	})
	if err != nil {
		log.Warnf("prefixDataFetch: Failed to express Interest: %+v", err)
		return
	}
}
