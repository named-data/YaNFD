package cmd

import (
	"errors"

	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/dv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

type DvConfig struct {
	// NFD related options
	Nfd struct {
		Unix string `json:"unix"`
	} `json:"nfd"`

	// Underlying configuration
	Config *config.Config `json:"config"`
}

func DefaultConfig() DvConfig {
	dc := DvConfig{}
	dc.Nfd.Unix = "/var/run/nfd/nfd.sock"
	dc.Config = config.DefaultConfig()
	return dc
}

func (dc DvConfig) Parse() error {
	return dc.Config.Parse()
}

type DvExecutor struct {
	engine *basic_engine.Engine
	router *dv.Router
}

func NewDvExecutor(dc DvConfig) (*DvExecutor, error) {
	dve := new(DvExecutor)

	// Validate configuration sanity
	err := dc.Parse()
	if err != nil {
		return nil, errors.New("failed to validate dv config: " + err.Error())
	}

	// Start NDN engine
	face := basic_engine.NewStreamFace("unix", dc.Nfd.Unix, true)
	timer := basic_engine.NewTimer()
	dve.engine = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), dve.noValidate)

	// Create the DV router
	dve.router, err = dv.NewRouter(dc.Config, dve.engine)
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
