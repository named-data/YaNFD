package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

var app *basic_engine.Engine

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func onInterest(
	interest ndn.Interest, rawInterest enc.Wire, sigCovered enc.Wire, reply ndn.ReplyFunc, deadline time.Time,
) {
	fmt.Printf(">> I: %s\n", interest.Name().String())
	content := []byte("Hello, world!")
	wire, _, err := app.Spec().MakeData(
		interest.Name(),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(10 * time.Second),
		},
		enc.Wire{content},
		sec.NewSha256Signer())
	if err != nil {
		log.WithField("module", "main").Errorf("unable to encode data: %+v", err)
		return
	}
	err = reply(wire)
	if err != nil {
		log.WithField("module", "main").Errorf("unable to reply with data: %+v", err)
		return
	}
	fmt.Printf("<< D: %s\n", interest.Name().String())
	fmt.Printf("ontent: (size: %d)\n", len(content))
	fmt.Printf("\n")
}

func main() {
	timer := basic_engine.NewTimer()
	// face := basic_engine.NewWebSocketFace("ws", "localhost:9696", true)
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd.sock", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")
	err := app.Start()
	if err != nil {
		logger.Errorf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	prefix, _ := enc.NameFromStr("/example/testApp")
	err = app.AttachHandler(prefix, onInterest)
	if err != nil {
		logger.Errorf("Unable to register handler: %+v", err)
		return
	}
	err = app.RegisterRoute(prefix)
	if err != nil {
		logger.Errorf("Unable to register route: %+v", err)
		return
	}

	fmt.Print("Start serving ...")
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	logger.Infof("Received signal %+v - exiting", receivedSig)
}
