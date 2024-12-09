package main

import (
	"os"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/engine/sync"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

var app *basic_engine.Engine

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <nodeId>", os.Args[0])
	}

	// Parse command line arguments
	nodeId, err := enc.NameFromStr(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid node ID: %s", os.Args[1])
	}

	// Create a new engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd/nfd.sock", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")
	err = app.Start()
	if err != nil {
		logger.Errorf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Start SVS instance
	group, _ := enc.NameFromStr("/ndn/svs")
	svsync := sync.NewSvSync(app, group)
	err = svsync.Start()
	if err != nil {
		logger.Errorf("Unable to create SvSync: %+v", err)
		return
	}

	// Publish new sequence number every second
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			new := svsync.IncrSeqNo(nodeId)
			logger.Infof("Published new sequence number: %d", new)
		case update := <-svsync.Updates:
			logger.Infof("Received update: %+v", update)
		}
	}
}
