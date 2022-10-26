package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

var app *basic_engine.Engine
var tree *schema.Tree

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func onInterest(matching enc.Matching, appParam enc.Wire, reply ndn.ReplyFunc, context schema.Context) bool {
	fmt.Printf(">> I: timestamp: %d\n", matching["time"].(uint64))
	content := []byte("Hello, world!")
	name := context[schema.CkName].(enc.Name)
	wire, _, err := app.Spec().MakeData(
		name,
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(10 * time.Second),
		},
		enc.Wire{content},
		sec.NewSha256Signer())
	if err != nil {
		log.WithField("module", "main").Errorf("unable to encode data: %+v", err)
		return true
	}
	err = reply(wire)
	if err != nil {
		log.WithField("module", "main").Errorf("unable to reply with data: %+v", err)
		return true
	}
	fmt.Printf("<< D: %s\n", name.String())
	fmt.Printf("ontent: (size: %d)\n", len(content))
	fmt.Printf("\n")
	return true
}

func onAttach(path enc.NamePattern, engine ndn.Engine) error {
	prefix := make(enc.Name, len(path))
	for i, p := range path {
		c, ok := p.(enc.Component)
		if !ok {
			return errors.New("unable to register a prefix that has unknown pattern")
		}
		prefix[i] = c
	}
	return engine.RegisterRoute(prefix)
}

func main() {
	// Setup engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd.sock", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")
	err := app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Setup schema tree
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/randomData/<t=time>")
	node := &schema.ExpressPoint{}
	err = tree.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	node.Set(schema.PropCanBePrefix, false)
	node.Set(schema.PropMustBeFresh, true)
	node.Set(schema.PropLifetime, 6*time.Second)
	node.Get(schema.PropOnInterest).(*schema.Event[*schema.NodeOnIntEvent]).Add(utils.IdPtr(onInterest))
	// The prefix registered is at the root
	tree.Root.Get(schema.PropOnAttach).(*schema.Event[*schema.NodeOnAttachEvent]).Add(utils.IdPtr(onAttach))
	prefix, _ := enc.NameFromStr("/example/testApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	fmt.Print("Start serving ...")
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	logger.Infof("Received signal %+v - exiting", receivedSig)
}
