package table

import (
	"time"

	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/nfdc"
	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type NeighborTable struct {
	// main DV config
	config *config.Config
	// nfd management thread
	nfdc *nfdc.NfdMgmtThread
	// neighbor name hash -> neighbor
	neighbors map[uint64]*NeighborState
}

type NeighborState struct {
	// neighbor name
	Name enc.Name
	// advertisement sequence number for neighbor
	AdvertSeq uint64
	// most recent advertisement
	Advert *tlv.Advertisement

	// pointer to the neighbor table
	nt *NeighborTable
	// time of last sync interest
	lastSeen time.Time
	// list of faces that are registered to this neighbor
	faces []*NeighborFace
	// most recent advertisement (wire)
	advertWire []byte
}

type NeighborFace struct {
	// face ID
	faceId uint64
	// time of last sync interest
	lastSeen time.Time
}

func NewNeighborTable(config *config.Config, nfdc *nfdc.NfdMgmtThread) *NeighborTable {
	return &NeighborTable{
		config:    config,
		nfdc:      nfdc,
		neighbors: make(map[uint64]*NeighborState),
	}
}

func (nt *NeighborTable) Get(name enc.Name) *NeighborState {
	return nt.GetH(name.Hash())
}

func (nt *NeighborTable) GetH(nameHash uint64) *NeighborState {
	neighbor, ok := nt.neighbors[nameHash]
	if !ok {
		return nil
	}
	return neighbor
}

func (nt *NeighborTable) Add(name enc.Name) *NeighborState {
	neighbor := &NeighborState{
		Name: name.Clone(),
		nt:   nt,
	}
	nt.neighbors[name.Hash()] = neighbor
	return neighbor
}

func (nt *NeighborTable) Remove(name enc.Name) {
	hash := name.Hash()
	if ns := nt.GetH(hash); ns != nil {
		ns.Advert = nil
	}
	delete(nt.neighbors, hash)
}

func (nt *NeighborTable) GetAll() []*NeighborState {
	neighbors := make([]*NeighborState, 0, len(nt.neighbors))
	for _, neighbor := range nt.neighbors {
		neighbors = append(neighbors, neighbor)
	}
	return neighbors
}

func (ns *NeighborState) SetAdvert(advert *tlv.Advertisement, wire []byte) {
	ns.Advert = advert
	ns.advertWire = wire
}

// Call this when a ping is received from a face.
// This will automatically register the face route with the neighbor
// and update the last seen time for the neighbor.
func (ns *NeighborState) RecvPing(faceId uint64) error {
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
	ns.faces = append(ns.faces, &NeighborFace{
		faceId:   faceId,
		lastSeen: ns.lastSeen,
	})

	ns.nt.nfdc.Exec(nfdc.NfdMgmtCmd{
		Module: "rib",
		Cmd:    "register",
		Args: &mgmt.ControlArgs{
			FaceId: utils.IdPtr(faceId),
			Name:   ns.Name,
		},
		Retries: 8,
	})

	return nil
}

func (ns *NeighborState) Prune() {
	// Prune faces that have not been seen in a while
	for _, face := range ns.faces {
		if time.Since(face.lastSeen) > ns.nt.config.RouterDeadInterval {
			ns.nt.nfdc.Exec(nfdc.NfdMgmtCmd{
				Module: "rib",
				Cmd:    "unregister",
				Args: &mgmt.ControlArgs{
					FaceId: utils.IdPtr(face.faceId),
					Name:   ns.Name,
				},
				Retries: 3,
			})
		}
	}
}

func (ns *NeighborState) IsDead() bool {
	return time.Since(ns.lastSeen) > ns.nt.config.RouterDeadInterval
}
