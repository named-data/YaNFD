package dv

import (
	"time"

	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

func (dv *Router) advertDataFetch(nodeId enc.Name, seqNo uint64) {
	// debounce; wait before fetching, then check if this is still the latest
	// sequence number known for this neighbor
	time.Sleep(10 * time.Millisecond)
	if ns := dv.neighbors.Get(nodeId); ns == nil || ns.AdvertSeq != seqNo {
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
		log.Warnf("advertDataFetch: Failed to make Interest: %+v", err)
		return
	}

	// Fetch the advertisement
	err = dv.engine.Express(finalName, cfg, wire, func(
		result ndn.InterestResult, data ndn.Data,
		_ enc.Wire, _ enc.Wire, _ uint64,
	) {
		go func() { // Don't block the main loop
			if result != ndn.InterestResultData {
				// If this wasn't a timeout, wait for 2s before retrying
				// This prevents excessive retries in case of NACKs
				if result != ndn.InterestResultTimeout {
					time.Sleep(2 * time.Second)
				} else {
					time.Sleep(100 * time.Millisecond)
				}

				// Keep retrying until we get the advertisement
				// If the router is dead, we break out of this by checking
				// that the sequence number is gone (above)
				log.Warnf("advertDataFetch: Retrying %s: %+v", finalName.String(), result)
				dv.advertDataFetch(nodeId, seqNo)
				return
			}

			// Process the advertisement
			dv.advertDataHandler(data)
		}()
	})
	if err != nil {
		log.Warnf("advertDataFetch: Failed to express Interest: %+v", err)
	}
}

func (dv *Router) advertDataOnInterestAsync(
	interest ndn.Interest,
	reply ndn.ReplyFunc,
	extra ndn.InterestHandlerExtra,
) {
	go dv.advertDataOnInterest(interest, reply, extra)
}

// Received advertisement Interest
func (dv *Router) advertDataOnInterest(
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
		return dv.rib.Advert()
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
		log.Warnf("advertDataOnInterest: Failed to make Data: %+v", err)
		return
	}

	// Send the Data packet
	err = reply(wire)
	if err != nil {
		log.Warnf("advertDataOnInterest: Failed to reply: %+v", err)
		return
	}
}

// Received advertisement Data
func (dv *Router) advertDataHandler(data ndn.Data) {
	// Parse name components
	name := data.Name()
	neighbor := name[:len(name)-3]
	seqNo := name[len(name)-1].NumberVal()

	// Lock DV state
	dv.mutex.Lock()
	defer dv.mutex.Unlock()

	// Check if this is the latest advertisement
	ns := dv.neighbors.Get(neighbor)
	if ns == nil || ns.AdvertSeq != seqNo {
		log.Warnf("advertDataHandler: Received old Advert for %s (%d != %d)", neighbor.String(), ns.AdvertSeq, seqNo)
		return
	}

	// TODO: verify signature on Advertisement
	log.Infof("advertDataHandler: Received: %s", data.Name().String())

	// Parse the advertisement
	raw := data.Content().Join() // clone
	advert, err := tlv.ParseAdvertisement(enc.NewWireReader(enc.Wire{raw}), false)
	if err != nil {
		log.Warnf("advertDataHandler: Failed to parse Advert: %+v", err)
		return
	}

	// Update the local advertisement list
	ns.Advert = advert
	go dv.ribUpdate(ns)
}
