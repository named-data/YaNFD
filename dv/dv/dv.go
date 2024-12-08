package dv

import (
	"sync"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
)

type Config struct {
	// GlobalPrefix should be the same for all routers in the network.
	GlobalPrefix string
	// RouterPrefix should be unique for each router in the network.
	RouterPrefix string
}

type DV struct {
	// go-ndn app that this router is attached to
	engine *basic_engine.Engine
	// config for this router
	config *Config

	// channel to stop the DV
	stop chan bool
	// heartbeat for outgoing Advertisements
	heartbeat *time.Ticker
	// single mutex for all operations
	mutex sync.Mutex

	// global Prefix
	globalPrefix enc.Name
	// router Prefix
	routerPrefix enc.Name

	// advertisement sequence number for self
	advSeq uint64
	// advertisement sequence numbers for neighbors
	neighborAdvSeq map[uint64]uint64
}

// Create a new DV router.
func NewDV(config *Config, engine *basic_engine.Engine) *DV {
	return &DV{
		engine: engine,
		config: config,

		stop:      make(chan bool),
		heartbeat: time.NewTicker(2 * time.Second), // TODO: configurable

		advSeq:         uint64(time.Now().UnixMilli()), // TODO: not efficient
		neighborAdvSeq: make(map[uint64]uint64),
	}
}

// Start the DV router. Blocks until Stop() is called.
func (dv *DV) Start() (err error) {
	// Register interest handlers
	// TODO: make this configurable
	err = dv.register()
	if err != nil {
		return err
	}

	for {
		select {
		case <-dv.heartbeat.C:
			dv.syncAdvertisement()
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
	// Parse prefixes
	dv.globalPrefix, err = enc.NameFromStr(dv.config.GlobalPrefix)
	if err != nil {
		return err
	}
	dv.routerPrefix, err = enc.NameFromStr(dv.config.RouterPrefix)
	if err != nil {
		return err
	}

	// Advertisement Sync
	prefixAdvSync := append(dv.globalPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
	)
	err = dv.engine.AttachHandler(prefixAdvSync, dv.onAdvSyncInterest)
	if err != nil {
		return err
	}

	// Advertisement Data
	prefixAdv := append(dv.routerPrefix,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
	)
	err = dv.engine.AttachHandler(prefixAdv, dv.onAdvInterest)
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
