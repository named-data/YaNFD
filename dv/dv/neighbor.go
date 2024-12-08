package dv

import (
	"time"

	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type neighbor_state struct {
	// neighbor name
	name enc.Name
	// time of last sync interest
	lastSeen time.Time
	// list of faces that are registered to this neighbor
	faces []*neighbor_face
	// advertisement sequence number for neighbor
	advertSeq uint64
	// most recent advertisement
	advert *tlv.Advertisement
	// most recent advertisement (wire)
	advertWire []byte
}

type neighbor_face struct {
	// face ID
	faceId uint64
	// time of last sync interest
	lastSeen time.Time
}

// Call this when a ping is received from a face.
func (dv *DV) neighborPing(ns *neighbor_state, faceId uint64) error {
	// Update last seen time for neighbor
	ns.lastSeen = time.Now()

	// Check if already registered.
	// This likely has only one entry; iterating is faster than a map.
	for _, face := range ns.faces {
		if face.faceId == faceId {
			face.lastSeen = ns.lastSeen
			return nil
		}
	}

	// Mark face as registered
	ns.faces = append(ns.faces, &neighbor_face{
		faceId:   faceId,
		lastSeen: ns.lastSeen,
	})

	// TODO: retry on failures
	dv.mgmt.Exec(mgmt_cmd{
		module: "rib",
		cmd:    "register",
		args: &mgmt.ControlArgs{
			FaceId: utils.IdPtr(faceId),
			Name:   ns.name,
		},
		retries: 8,
	})

	return nil
}

func (dv *DV) neighborPrune(ns *neighbor_state) {
	// Prune faces that have not been seen in a while
	for _, face := range ns.faces {
		if time.Since(face.lastSeen) > dv.config.RouterDeadInterval {
			// TODO: retry on failures
			dv.mgmt.Exec(mgmt_cmd{
				module: "rib",
				cmd:    "unregister",
				args: &mgmt.ControlArgs{
					FaceId: utils.IdPtr(face.faceId),
					Name:   ns.name,
				},
				retries: 3,
			})
		}
	}
}

func (dv *DV) neighborIsDead(ns *neighbor_state) bool {
	return time.Since(ns.lastSeen) > dv.config.RouterDeadInterval
}
