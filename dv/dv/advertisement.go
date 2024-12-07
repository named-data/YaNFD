package dv

import (
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema/svs"
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

func (dv *DV) onAdvertisementSyncInterest(
	interest ndn.Interest,
	rawInterest enc.Wire,
	sigCovered enc.Wire,
	reply ndn.ReplyFunc,
	deadline time.Time,
) {
	fmt.Println("Received Sync Interest")
}

// Global Interest handler
func (dv *DV) onAdvertisementInterest(
	interest ndn.Interest,
	rawInterest enc.Wire,
	sigCovered enc.Wire,
	reply ndn.ReplyFunc,
	deadline time.Time,
) {
	fmt.Println("Received Interest")
}
