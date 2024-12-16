package cmd

import (
	"time"

	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/dv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

type YamlConfig struct {
	// Same as config.Config for parsing
	Config struct {
		NetworkName                  string `json:"network"`
		RouterName                   string `json:"router"`
		AdvertisementSyncInterval_ms uint64 `json:"advertise_interval"`
		RouterDeadInterval_ms        uint64 `json:"router_dead_interval"`
	} `json:"config"`

	// NFD related options
	Nfd struct {
		Unix string `json:"unix"`
	} `json:"nfd"`
}

func (yc YamlConfig) Parse() (*config.Config, error) {
	// Set configuration defaults
	if yc.Nfd.Unix == "" {
		yc.Nfd.Unix = "/var/run/nfd/nfd.sock"
	}
	if yc.Config.AdvertisementSyncInterval_ms == 0 {
		yc.Config.AdvertisementSyncInterval_ms = 5000
	}
	if yc.Config.RouterDeadInterval_ms == 0 {
		yc.Config.RouterDeadInterval_ms = 30000
	}

	// Convert to config.Config
	out := &config.Config{
		NetworkName:               yc.Config.NetworkName,
		RouterName:                yc.Config.RouterName,
		AdvertisementSyncInterval: time.Duration(yc.Config.AdvertisementSyncInterval_ms * uint64(time.Millisecond)),
		RouterDeadInterval:        time.Duration(yc.Config.RouterDeadInterval_ms * uint64(time.Millisecond)),
	}

	// Validate configuration
	err := out.Parse()
	if err != nil {
		return nil, err
	}

	return out, err
}

func noValidate(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func Run(yc YamlConfig) {
	// Validate configuration sanity
	cfg, err := yc.Parse()
	if err != nil {
		panic(err)
	}

	// Start NDN app
	face := basic_engine.NewStreamFace("unix", yc.Nfd.Unix, true)
	timer := basic_engine.NewTimer()
	app := basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), noValidate)

	// Enable logging
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")

	// Start the app
	err = app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Create the DV router
	router, err := dv.NewRouter(cfg, app)
	if err != nil {
		logger.Fatalf("Unable to create DV router: %+v", err)
		return
	}

	// Start the DV router
	err = router.Start()
	if err != nil {
		logger.Fatalf("Unable to start DV router: %+v", err)
		return
	}
	defer router.Stop()
}
