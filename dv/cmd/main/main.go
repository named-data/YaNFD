package main

import (
	"os"
	"time"

	"github.com/goccy/go-yaml"
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
		Unix string `json:"Unix"`
	} `json:"nfd"`
}

func noValidate(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	var cfgFile string = "/etc/ndn/dv.yml"
	if len(os.Args) >= 2 {
		cfgFile = os.Args[1]
	}

	// Configuration defaults
	yc := YamlConfig{}
	yc.Nfd.Unix = "/var/run/nfd/nfd.sock"
	yc.Config.AdvertisementSyncInterval_ms = 5000
	yc.Config.RouterDeadInterval_ms = 30000

	// Read configuration file
	cfgBytes, err := os.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	// Parse YAML configuration file
	if err = yaml.Unmarshal(cfgBytes, &yc); err != nil {
		panic(err)
	}

	// Convert configuration to dv struct
	config := &config.Config{
		NetworkName:               yc.Config.NetworkName,
		RouterName:                yc.Config.RouterName,
		AdvertisementSyncInterval: time.Duration(yc.Config.AdvertisementSyncInterval_ms * uint64(time.Millisecond)),
		RouterDeadInterval:        time.Duration(yc.Config.RouterDeadInterval_ms * uint64(time.Millisecond)),
	}

	// Validate configuration sanity
	if err = config.Parse(); err != nil {
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
	router, err := dv.NewRouter(config, app)
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
