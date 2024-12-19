package table

import (
	"sync"
	"time"

	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/security"
	ndn_sync "github.com/zjkmxy/go-ndn/pkg/sync"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type PrefixTable struct {
	config *config.Config
	engine ndn.Engine
	svs    *ndn_sync.SvSync

	routers map[uint64]*PrefixTableRouter
	me      *PrefixTableRouter

	repo       map[uint64][]byte
	repoMutex  sync.RWMutex
	snapshotAt uint64
}

type PrefixTableRouter struct {
	Name     enc.Name
	Fetching bool
	Known    uint64
	Latest   uint64
	Prefixes map[uint64]*PrefixEntry
}

type PrefixEntry struct {
	Name enc.Name
}

func NewPrefixTable(
	config *config.Config,
	engine ndn.Engine,
	svs *ndn_sync.SvSync,
) *PrefixTable {
	pt := &PrefixTable{
		config: config,
		engine: engine,
		svs:    svs,

		routers: make(map[uint64]*PrefixTableRouter),
		me:      nil,

		repo:      make(map[uint64][]byte),
		repoMutex: sync.RWMutex{},
	}

	pt.me = pt.GetRouter(config.RouterName())
	pt.me.Known = svs.GetSeqNo(config.RouterName())
	pt.me.Latest = pt.me.Known
	pt.publishSnap()

	return pt
}

func (pt *PrefixTable) GetRouter(name enc.Name) *PrefixTableRouter {
	hash := name.Hash()
	router := pt.routers[hash]
	if router == nil {
		router = &PrefixTableRouter{
			Name:     name,
			Prefixes: make(map[uint64]*PrefixEntry),
		}
		pt.routers[hash] = router
	}
	return router
}

func (pt *PrefixTable) Announce(name enc.Name) {
	log.Infof("prefix-table: announcing %s", name)
	hash := name.Hash()

	// Skip if matching entry already exists
	// This will also need to check that all params are equal
	if entry := pt.me.Prefixes[hash]; entry != nil && entry.Name.Equal(name) {
		return
	}

	// Create new entry and announce globally
	pt.me.Prefixes[hash] = &PrefixEntry{Name: name}

	op := tlv.PrefixOpList{
		ExitRouter: &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpAdds: []*tlv.PrefixOpAdd{{
			Name: name,
			Cost: 1,
		}},
	}
	pt.publishOp(op.Encode())
}

func (pt *PrefixTable) Withdraw(name enc.Name) {
	log.Infof("prefix-table: withdrawing %s", name)
	hash := name.Hash()

	// Check if entry does not exist
	if entry := pt.me.Prefixes[hash]; entry == nil {
		return
	}

	// Delete the existing entry and announce globally
	delete(pt.me.Prefixes, hash)

	op := tlv.PrefixOpList{
		ExitRouter:      &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpRemoves: []*tlv.PrefixOpRemove{{Name: name}},
	}
	pt.publishOp(op.Encode())
}

// Applies ops from a list. Returns if dirty.
func (pt *PrefixTable) Apply(ops *tlv.PrefixOpList) (dirty bool) {
	if ops.ExitRouter == nil || len(ops.ExitRouter.Name) == 0 {
		log.Error("prefix-table: received PrefixOpList has no ExitRouter")
		return false
	}

	router := pt.GetRouter(ops.ExitRouter.Name)

	if ops.PrefixOpReset {
		log.Infof("prefix-table: reset prefix table for %s", ops.ExitRouter.Name)
		router.Prefixes = make(map[uint64]*PrefixEntry)
		dirty = true
	}

	for _, add := range ops.PrefixOpAdds {
		log.Infof("prefix-table: added prefix for %s: %s", ops.ExitRouter.Name, add.Name)
		router.Prefixes[add.Name.Hash()] = &PrefixEntry{Name: add.Name}
		dirty = true
	}

	for _, remove := range ops.PrefixOpRemoves {
		log.Infof("prefix-table: removed prefix for %s: %s", ops.ExitRouter.Name, remove.Name)
		delete(router.Prefixes, remove.Name.Hash())
		dirty = true
	}

	return dirty
}

func (pt *PrefixTable) publishOp(content enc.Wire) {
	// Increment our sequence number
	seq := pt.svs.IncrSeqNo(pt.config.RouterName())
	pt.me.Known = seq
	pt.me.Latest = seq

	// Create the new data
	name := append(pt.config.PrefixTableDataPrefix(), enc.NewSequenceNumComponent(seq))
	pt.publish(name, content)

	// Create snapshot if needed
	if pt.snapshotAt-seq >= 100 {
		pt.publishSnap()
	}
}

func (pt *PrefixTable) publishSnap() {
	snap := tlv.PrefixOpList{
		ExitRouter:    &tlv.Destination{Name: pt.config.RouterName()},
		PrefixOpReset: true,
		PrefixOpAdds:  make([]*tlv.PrefixOpAdd, 0, len(pt.me.Prefixes)),
	}

	for _, entry := range pt.me.Prefixes {
		snap.PrefixOpAdds = append(snap.PrefixOpAdds, &tlv.PrefixOpAdd{
			Name: entry.Name,
			Cost: 1,
		})
	}

	// Store snapshot in repo
	// TODO: this can be a segmented object
	pt.snapshotAt = pt.me.Latest
	snapPfx := append(pt.config.PrefixTableDataPrefix(),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "SNAP"))
	snapName := append(snapPfx, enc.NewSequenceNumComponent(pt.snapshotAt))
	pt.publish(snapName, snap.Encode())

	// Point prefix to the snapshot
	pt.repoMutex.Lock()
	defer pt.repoMutex.Unlock()
	pt.repo[snapPfx.Hash()] = pt.repo[snapName.Hash()]
}

func (pt *PrefixTable) publish(name enc.Name, content enc.Wire) {
	// TODO: sign the prefix table data
	signer := security.NewSha256Signer()

	data, err := pt.engine.Spec().MakeData(
		name,
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(1 * time.Second),
		},
		content,
		signer)
	if err != nil {
		log.Warnf("prefix-table: publish failed to make data: %+v", err)
		return
	}

	// Store the data packet in our mem repo
	pt.repoMutex.Lock()
	defer pt.repoMutex.Unlock()
	pt.repo[name.Hash()] = data.Wire.Join()
}

// Received prefix data Interest
func (pt *PrefixTable) OnDataInterest(args ndn.InterestHandlerArgs) {
	// TODO: remove old entries from repo

	pt.repoMutex.RLock()
	defer pt.repoMutex.RUnlock()

	// Find exact match in repo
	name := args.Interest.Name()
	if data := pt.repo[name.Hash()]; data != nil {
		err := args.Reply(enc.Wire{data})
		if err != nil {
			log.Warnf("prefix-table: failed to reply: %+v", err)
		}
		return
	}

	log.Warnf("prefix-table: repo failed to find data for for %s", name)
}
