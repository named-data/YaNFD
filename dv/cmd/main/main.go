package main

import (
	"os"
	"time"

	"github.com/pulsejet/go-ndn-dv/dv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

func noValidate(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	// Create face to forwarder
	sock := "/var/run/nfd/nfd.sock"
	if len(os.Args) > 3 {
		sock = os.Args[3]
	}
	face := basic_engine.NewStreamFace("unix", sock, true)

	// Start NDN app
	timer := basic_engine.NewTimer()
	app := basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), noValidate)

	// Enable logging
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")

	// Start the app
	err := app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Create a new DV router
	config := &dv.Config{
		GlobalPrefix:              os.Args[1],
		RouterPrefix:              os.Args[2],
		AdvertisementSyncInterval: 2 * time.Second,
		RouterDeadInterval:        5 * time.Second,
	}

	router, err := dv.NewDV(config, app)
	if err != nil {
		logger.Fatalf("Unable to create DV router: %+v", err)
		return
	}

	err = router.Start()
	if err != nil {
		logger.Fatalf("Unable to start DV router: %+v", err)
		return
	}
}
