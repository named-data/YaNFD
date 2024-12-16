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
	// pointer to the neighbor table
	nt *NeighborTable

	// neighbor name
	Name enc.Name
	// advertisement sequence number for neighbor
	AdvertSeq uint64
	// most recent advertisement
	Advert *tlv.Advertisement

	// time of last sync interest
	lastSeen time.Time
	// latest known face ID
	faceId uint64
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
	return nt.neighbors[nameHash]
}

func (nt *NeighborTable) Add(name enc.Name) *NeighborState {
	neighbor := &NeighborState{
		nt: nt,

		Name:      name.Clone(),
		AdvertSeq: 0,
		Advert:    nil,

		lastSeen: time.Now(),
		faceId:   0,
	}
	nt.neighbors[name.Hash()] = neighbor
	return neighbor
}

func (nt *NeighborTable) Remove(name enc.Name) {
	hash := name.Hash()
	if ns := nt.GetH(hash); ns != nil {
		ns.delete()
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

func (ns *NeighborState) IsDead() bool {
	return time.Since(ns.lastSeen) > ns.nt.config.RouterDeadInterval
}

// Call this when a ping is received from a face.
// This will automatically register the face route with the neighbor
// and update the last seen time for the neighbor.
func (ns *NeighborState) RecvPing(faceId uint64) error {
	// Update last seen time for neighbor
	ns.lastSeen = time.Now()

	// If face ID has changed, re-register face.
	if ns.faceId != faceId {
		ns.routeUnregister()
		ns.routeRegister(faceId)
	}

	return nil
}

// Called when the neighbor is removed from the neighbor table.
func (ns *NeighborState) delete() {
	ns.Advert = nil
	ns.routeUnregister()
}

func (ns *NeighborState) route() enc.Name {
	return append(ns.Name, enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"))
}

// Register route to this neighbor
func (ns *NeighborState) routeRegister(faceId uint64) {
	ns.faceId = faceId
	ns.nt.nfdc.Exec(nfdc.NfdMgmtCmd{
		Module: "rib",
		Cmd:    "register",
		Args: &mgmt.ControlArgs{
			Name:   ns.route(),
			FaceId: utils.IdPtr(faceId),
			Origin: utils.IdPtr(config.NlsrOrigin),
			Cost:   utils.IdPtr(uint64(0)),
		},
		Retries: 3,
	})
}

// Single attempt to unregister the route
func (ns *NeighborState) routeUnregister() {
	if ns.faceId == 0 {
		return // not set
	}
	ns.nt.nfdc.Exec(nfdc.NfdMgmtCmd{
		Module: "rib",
		Cmd:    "unregister",
		Args: &mgmt.ControlArgs{
			Name:   ns.route(),
			FaceId: utils.IdPtr(ns.faceId),
			Origin: utils.IdPtr(config.NlsrOrigin),
		},
		Retries: 1,
	})
}
