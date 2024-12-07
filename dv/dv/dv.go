package dv

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
)

type DV struct {
	// go-ndn app that this router is attached to
	engine *basic_engine.Engine
	// channel to stop the DV
	stop chan bool
	// heartbeat for outgoing Advertisements
	heartbeat *time.Ticker
}

// Create a new DV router.
func NewDV(engine *basic_engine.Engine) *DV {
	return &DV{
		engine:    engine,
		stop:      make(chan bool),
		heartbeat: time.NewTicker(5 * time.Second), // TODO: configurable
	}
}

// Start the DV router. Blocks until Stop() is called.
func (dv *DV) Start() (err error) {
	// Register interest handlers
	// TODO: make this configurable
	err = dv.register("/ndn", "/router1")
	if err != nil {
		return err
	}

	for {
		select {
		case <-dv.heartbeat.C:
			dv.Advertise()
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
func (dv *DV) register(
	globalPrefix string,
	routerPrefix string,
) (err error) {
	// Parse prefixes
	globalPrefixN, err := enc.NameFromStr(globalPrefix)
	if err != nil {
		return err
	}
	routerPrefixN, err := enc.NameFromStr(routerPrefix)
	if err != nil {
		return err
	}

	// Advertisement Data
	advPrefix := append(routerPrefixN,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
	)
	err = dv.engine.AttachHandler(advPrefix, dv.onAdvertisementInterest)
	if err != nil {
		return err
	}

	// Advertisement Sync
	advSyncPrefix := append(globalPrefixN,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
	)
	err = dv.engine.AttachHandler(advSyncPrefix, dv.onAdvertisementSyncInterest)
	if err != nil {
		return err
	}

	// Register routes to forwarder
	for _, prefix := range []enc.Name{advPrefix, advSyncPrefix} {
		err = dv.engine.RegisterRoute(prefix)
		if err != nil {
			return err
		}
	}

	return nil
}
