package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
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
	node.Set("AggregateInterval", 100*time.Millisecond)

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

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(2)
	// 1. Randomly produce data
	ticker := time.Tick(5 * time.Second)
	go func() {
		defer wg.Done()
		for val := 0; true; val++ {
			select {
			case <-ticker:
				text := fmt.Sprintf("[%s: TICK %d]\n", nodeId, val)
				node.NewData(enc.Wire{[]byte(text)}, schema.Context{})
				fmt.Printf("Produced: %s", text)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 2. On data received, print
	go func() {
		defer wg.Done()
		ch := node.MissingDataChannel()
		for {
			select {
			case missData := <-ch:
				for i := missData.StartSeq; i < missData.EndSeq; i++ {
					ret, data := (<-node.Need(missData.NodeId, i, enc.Matching{}, schema.Context{})).Get()
					if ret != ndn.InterestResultData {
						fmt.Printf("Data fetching failed for (%s, %d): %+v\n", string(missData.NodeId), i, ret)
					} else {
						fmt.Printf("Fetched (%s, %d): %s", string(missData.NodeId), i, string(data.Join()))
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for keyboard quit signal
	sigChannel := make(chan os.Signal, 1)
	fmt.Print("Start serving ...\n")
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	logger.Infof("Received signal %+v - exiting\n", receivedSig)
	cancel()
	wg.Wait()
}
