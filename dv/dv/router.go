package dv

import (
	"sync"
	"time"

	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/nfdc"
	"github.com/pulsejet/go-ndn-dv/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type Router struct {
	// go-ndn app that this router is attached to
	engine *basic_engine.Engine
	// nfd management thread
	nfdc *nfdc.NfdMgmtThread
	// single mutex for all operations
	mutex sync.Mutex

	// config for this router
	config *config.Config
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

	// neighbor table
	neighbors *table.NeighborTable
	// routing information base
	rib *table.Rib

	// advertisement sequence number for self
	advertSyncSeq uint64
}

// Create a new DV router.
func NewRouter(config *config.Config, engine *basic_engine.Engine) (*Router, error) {
	// Validate and parse configuration
	globalPrefix, err := enc.NameFromStr(config.GlobalPrefix)
	if err != nil {
		return nil, err
	}

	routerPrefix, err := enc.NameFromStr(config.RouterPrefix)
	if err != nil {
		return nil, err
	}

	// Create the NFD management thread
	nfdc := nfdc.NewNfdMgmtThread(engine)

	// Create the DV router
	return &Router{
		engine: engine,
		nfdc:   nfdc,
		mutex:  sync.Mutex{},

		config:       config,
		globalPrefix: globalPrefix,
		routerPrefix: routerPrefix,

		neighbors: table.NewNeighborTable(config, nfdc),
		rib:       table.NewRib(config),

		advertSyncSeq: uint64(time.Now().UnixMilli()),
	}, nil
}

// Start the DV router. Blocks until Stop() is called.
func (dv *Router) Start() (err error) {
	dv.stop = make(chan bool)

	// Start timers
	dv.heartbeat = time.NewTicker(dv.config.AdvertisementSyncInterval)
	dv.deadcheck = time.NewTicker(dv.config.RouterDeadInterval)
	defer dv.heartbeat.Stop()
	defer dv.deadcheck.Stop()

	// Start management thread
	go dv.nfdc.Start()
	defer dv.nfdc.Stop()

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
	dv.rib.Set(dv.routerPrefix, dv.routerPrefix, 0)

	for {
		select {
		case <-dv.heartbeat.C:
			dv.advertSyncSendInterest()
		case <-dv.deadcheck.C:
			dv.checkDeadNeighbors()
		case <-dv.stop:
			return nil
		}
	}
}

// Stop the DV router.
func (dv *Router) Stop() {
	dv.stop <- true
}

// Configure the face to forwarder.
func (dv *Router) configureFace() (err error) {
	// Enable local fields on face. This includes incoming face indication.
	dv.nfdc.Exec(nfdc.NfdMgmtCmd{
		Module: "faces",
		Cmd:    "update",
		Args: &mgmt.ControlArgs{
			Mask:  utils.IdPtr(uint64(0x01)),
			Flags: utils.IdPtr(uint64(0x01)),
		},
		Retries: -1,
	})

	return nil
}

// Register interest handlers for DV prefixes.
func (dv *Router) register() (err error) {
	// TODO: retry when these fail

	// Advertisement Sync
	prefixAdvSync := append(dv.globalPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
	)
	err = dv.engine.AttachHandler(prefixAdvSync, dv.advertSyncOnInterest)
	if err != nil {
		return err
	}

	// Advertisement Data
	prefixAdv := append(dv.routerPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
	)
	err = dv.engine.AttachHandler(prefixAdv, dv.advertDataOnInterest)
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
	mcast, _ := enc.NameFromStr(config.MulticastStrategy)
	pfxs = []enc.Name{
		prefixAdvSync,
	}
	for _, prefix := range pfxs {
		dv.nfdc.Exec(nfdc.NfdMgmtCmd{
			Module: "strategy-choice",
			Cmd:    "set",
			Args: &mgmt.ControlArgs{
				Name: prefix,
				Strategy: &mgmt.Strategy{
					Name: mcast,
				},
			},
			Retries: -1,
		})
	}

	return nil
}
