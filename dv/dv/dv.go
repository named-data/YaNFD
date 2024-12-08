package dv

import (
	"sync"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

const CostInfinity = uint64(16)
const MulticastStrategy = "/localhost/nfd/strategy/multicast"

type Config struct {
	// GlobalPrefix should be the same for all routers in the network.
	GlobalPrefix string
	// RouterPrefix should be unique for each router in the network.
	RouterPrefix string
	// Period of sending Advertisement Sync Interests.
	AdvertisementSyncInterval time.Duration
	// Time after which a neighbor is considered dead.
	RouterDeadInterval time.Duration
}

type DV struct {
	// go-ndn app that this router is attached to
	engine *basic_engine.Engine
	// single mutex for all operations
	mutex sync.Mutex

	// config for this router
	config *Config
	// global Prefix
	globalPrefix enc.Name
	// router Prefix
	routerPrefix enc.Name

	// channel to stop the DV
	stop chan bool
	// heartbeat for outgoing Advertisements
	heartbeat *time.Ticker
	// deadcheck for neighbors
	deadcheck *time.Ticker

	// advertisement sequence number for self
	advertSeq uint64
	// routing information base
	rib *rib
	// state of our neighbors
	// neighbor name hash -> neighbor
	neighbors map[uint64]*neighbor_state

	// management thread
	mgmt *mgmt_thread
}

// Create a new DV router.
func NewDV(config *Config, engine *basic_engine.Engine) (*DV, error) {
	// Validate and parse configuration
	globalPrefix, err := enc.NameFromStr(config.GlobalPrefix)
	if err != nil {
		return nil, err
	}

	routerPrefix, err := enc.NameFromStr(config.RouterPrefix)
	if err != nil {
		return nil, err
	}

	// Create the DV router
	return &DV{
		engine: engine,

		config:       config,
		globalPrefix: globalPrefix,
		routerPrefix: routerPrefix,

		advertSeq: uint64(time.Now().UnixMilli()), // TODO: not efficient
		rib:       NewRib(),
		neighbors: make(map[uint64]*neighbor_state),

		mgmt: newMgmtThread(engine),
	}, nil
}

// Start the DV router. Blocks until Stop() is called.
func (dv *DV) Start() (err error) {
	dv.stop = make(chan bool)

	// Start timers
	dv.heartbeat = time.NewTicker(dv.config.AdvertisementSyncInterval)
	dv.deadcheck = time.NewTicker(dv.config.RouterDeadInterval)
	defer dv.heartbeat.Stop()
	defer dv.deadcheck.Stop()

	// Start management thread
	go dv.mgmt.Start()
	defer dv.mgmt.Stop()

	// Configure face
	err = dv.configureFace()
	if err != nil {
		return err
	}

	// Register interest handlers
	err = dv.register()
	if err != nil {
		return err
	}

	// Add self to the RIB
	dv.rib.set(dv.routerPrefix, dv.routerPrefix, 0)

	for {
		select {
		case <-dv.heartbeat.C:
			dv.sendAdvertSyncInterest()
		case <-dv.deadcheck.C:
			dv.checkDeadNeighbors()
		case <-dv.stop:
			return nil
		}
	}
}

// Stop the DV router.
func (dv *DV) Stop() {
	dv.stop <- true
}

// Configure the face to forwarder.
func (dv *DV) configureFace() (err error) {
	// TODO: retry when these fail

	// Enable local fields on face. This includes incoming face indication.
	dv.mgmt.Exec(mgmt_cmd{
		module: "faces",
		cmd:    "update",
		args: &mgmt.ControlArgs{
			Mask:  utils.IdPtr(uint64(0x01)),
			Flags: utils.IdPtr(uint64(0x01)),
		},
		retries: 0,
	})

	return nil
}

// Register interest handlers for DV prefixes.
func (dv *DV) register() (err error) {
	// TODO: retry when these fail

	// Advertisement Sync
	prefixAdvSync := append(dv.globalPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
	)
	err = dv.engine.AttachHandler(prefixAdvSync, dv.onAdvertSyncInterest)
	if err != nil {
		return err
	}

	// Advertisement Data
	prefixAdv := append(dv.routerPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
	)
	err = dv.engine.AttachHandler(prefixAdv, dv.onAdvertInterest)
	if err != nil {
		return err
	}

	// Register routes to forwarder
	pfxs := []enc.Name{
		prefixAdv,
		prefixAdvSync,
	}
	for _, prefix := range pfxs {
		err = dv.engine.RegisterRoute(prefix)
		if err != nil {
			return err
		}
	}

	// Set strategy to multicast for sync prefixes
	mcast, _ := enc.NameFromStr(MulticastStrategy)
	pfxs = []enc.Name{
		prefixAdvSync,
	}
	for _, prefix := range pfxs {
		dv.mgmt.Exec(mgmt_cmd{
			module: "strategy-choice",
			cmd:    "set",
			args: &mgmt.ControlArgs{
				Name: prefix,
				Strategy: &mgmt.Strategy{
					Name: mcast,
				},
			},
		})
	}

	return nil
}
