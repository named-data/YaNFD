package cmd

import (
	"errors"
	"time"

	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/dv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
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

func DefaultConfig() YamlConfig {
	yc := YamlConfig{}
	yc.Nfd.Unix = "/var/run/nfd/nfd.sock"
	yc.Config.AdvertisementSyncInterval_ms = 5000
	yc.Config.RouterDeadInterval_ms = 30000
	return yc
}

func (yc YamlConfig) Parse() (*config.Config, error) {
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

type DvExecutor struct {
	engine *basic_engine.Engine
	router *dv.Router
}

func NewDvExecutor(yc YamlConfig) (*DvExecutor, error) {
	dve := new(DvExecutor)

	// Validate configuration sanity
	cfg, err := yc.Parse()
	if err != nil {
		return nil, errors.New("failed to validate dv config: " + err.Error())
	}

	// Start NDN engine
	face := basic_engine.NewStreamFace("unix", yc.Nfd.Unix, true)
	timer := basic_engine.NewTimer()
	dve.engine = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), dve.noValidate)

	// Create the DV router
	dve.router, err = dv.NewRouter(cfg, dve.engine)
	if err != nil {
		return nil, errors.New("failed to create dv router: " + err.Error())
	}

	return dve, nil
}

func (dve *DvExecutor) Start() error {
	err := dve.engine.Start()
	if err != nil {
		return errors.New("failed to start dv app: " + err.Error())
	}
	defer dve.engine.Shutdown()

	err = dve.router.Start() // blocks forever
	if err != nil {
		return errors.New("failed to start dv router: " + err.Error())
	}

	return nil
}

func (dve *DvExecutor) Stop() {
	dve.router.Stop()
}

func (dve *DvExecutor) Router() *dv.Router {
	return dve.router
}

func (dve *DvExecutor) noValidate(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}
