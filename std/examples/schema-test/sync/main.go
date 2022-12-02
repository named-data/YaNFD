package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/demo"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

const HmacKey = "Hello, World!"

var app *basic_engine.Engine
var tree *schema.Tree

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	// Note: remember to ` nfdc strategy set /example/schema /localhost/nfd/strategy/multicast `
	log.SetLevel(log.ErrorLevel)
	logger := log.WithField("module", "main")
	rand.Seed(time.Now().UnixMicro())

	// Setup schema tree
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/sync")
	node := &demo.SvsNode{}
	err := tree.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}
	nodeId := fmt.Sprintf("node-%d", rand.Int())
	fmt.Printf("Node ID: %s\n", nodeId)
	node.Set("SelfNodeId", []byte(nodeId))
	node.Set("ChannelSize", 1000)
	node.Set("SyncInterval", 2*time.Second)

	// Setup policies
	demo.NewFixedKeySigner([]byte(HmacKey)).Apply(node)
	demo.NewFixedKeyIntSigner([]byte(HmacKey)).Apply(node)
	demo.NewMemStoragePolicy().Apply(node)
	demo.NewRegisterPolicy().Apply(tree.Root) // TODO: This is not good; we shouldn't register for other node's prefix

	// Start engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd.sock", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	err = app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Attach schema
	prefix, _ := enc.NameFromStr("/example/schema/syncApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// 1. Randomly produce data
	go func() {
		val := 0
		for {
			val++
			time.Sleep(5 * time.Second)
			text := fmt.Sprintf("[%s: TICK %d]\n", nodeId, val)
			node.NewData(enc.Wire{[]byte(text)}, schema.Context{})
			fmt.Printf("Produced: %s", text)
		}
	}()

	// 2. On data received, print
	go func() {
		ch := node.MissingDataChannel()
		for {
			missData := <-ch
			for i := missData.StartSeq; i < missData.EndSeq; i++ {
				ret, data := (<-node.Need(missData.NodeId, i, enc.Matching{}, schema.Context{})).Get()
				if ret != ndn.InterestResultData {
					fmt.Printf("Data fetching failed for (%s, %d): %+v\n", string(missData.NodeId), i, ret)
				} else {
					fmt.Printf("Fetched (%s, %d): %s", string(missData.NodeId), i, string(data.Join()))
				}
			}
		}
	}()

	// Wait for keyboard quit signal
	sigChannel := make(chan os.Signal, 1)
	fmt.Print("Start serving ...\n")
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	logger.Infof("Received signal %+v - exiting\n", receivedSig)
}
