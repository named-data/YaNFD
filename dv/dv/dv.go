package dv

import (
	"sync"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
)

const CostInfinity = uint64(16)

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

		stop:      make(chan bool),
		heartbeat: time.NewTicker(config.AdvertisementSyncInterval),
		deadcheck: time.NewTicker(config.RouterDeadInterval),

		advertSeq: uint64(time.Now().UnixMilli()), // TODO: not efficient
		rib:       NewRib(),
		neighbors: make(map[uint64]*neighbor_state),
	}, nil
}

// Start the DV router. Blocks until Stop() is called.
func (dv *DV) Start() (err error) {
	// Add self to the RIB
	dv.rib.set(dv.routerPrefix, dv.routerPrefix, 0)

	// Register interest handlers
	err = dv.register()
	if err != nil {
		return err
	}

	// TODO: set strategy to multicast

	for {
		select {
		case <-dv.heartbeat.C:
			dv.sendAdvertSyncInterest()
		case <-dv.deadcheck.C:
			dv.checkDeadNeighbors()
		case <-dv.stop:
			return
		}
	}
}

// Stop the DV router.
func (dv *DV) Stop() {
	dv.heartbeat.Stop()
	dv.stop <- true
}

// Register interest handlers for DV prefixes.
func (dv *DV) register() (err error) {
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

	return nil
}
