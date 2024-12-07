package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	sec_pib "github.com/zjkmxy/go-ndn/pkg/security/pib"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

var app *basic_engine.Engine
var pib *sec_pib.SqlitePib

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func onInterest(
	interest ndn.Interest, rawInterest enc.Wire, sigCovered enc.Wire, reply ndn.ReplyFunc, deadline time.Time,
) {
	fmt.Printf(">> I: %s\n", interest.Name().String())
	content := []byte("Hello, world!")

	idName, _ := enc.NameFromStr("/test")
	identity := pib.GetIdentity(idName)
	cert := identity.FindCert(func(_ sec_pib.Cert) bool { return true })
	signer := cert.AsSigner()

	wire, _, err := app.Spec().MakeData(
		interest.Name(),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(10 * time.Second),
		},
		enc.Wire{content},
		signer)
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
	fmt.Printf("Content: (size: %d)\n", len(content))
	fmt.Printf("\n")
}

func main() {
	timer := basic_engine.NewTimer()
	// face := basic_engine.NewWebSocketFace("ws", "localhost:9696", true)
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd/nfd.sock", true)

	homedir, _ := os.UserHomeDir()
	tpm := sec_pib.NewFileTpm(filepath.Join(homedir, ".ndn/ndnsec-key-file"))
	pib = sec_pib.NewSqlitePib(filepath.Join(homedir, ".ndn/pib.db"), tpm)

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
