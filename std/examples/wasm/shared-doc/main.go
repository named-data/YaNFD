//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"syscall/js"
	"time"

	"github.com/apex/log"
	"github.com/zjkmxy/go-ndn/examples/schema-test/shared-doc/crdt"
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
var syncNode *demo.SvsNode
var textDoc crdt.TextDoc
var dataLock sync.Mutex
var nodeId string
var inputEle js.Value

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func onBeforeInput(this js.Value, args []js.Value) any {
	event := args[0]
	data := event.Get("data").String()
	inputType := event.Get("inputType").String()
	pos := inputEle.Get("selectionStart").Int()
	fmt.Printf("Event: %d, %s, %s\n", pos, data, inputType)

	if inputType == "insertText" || inputType == "deleteContentBackward" {
		dataLock.Lock()
		defer dataLock.Unlock()
		var rec *crdt.Record // the new record to send
		if inputType == "insertText" {
			rec = textDoc.Insert(int(pos), data)
		} else {
			rec = textDoc.Delete(int(pos))
		}
		if rec != nil {
			syncNode.NewData(rec.Encode(), schema.Context{})
		}
	}

	return nil
}

func main() {
	// Note: remember to ` nfdc strategy set /example/schema /localhost/nfd/strategy/multicast `
	log.SetLevel(log.DebugLevel)
	logger := log.WithField("module", "main")
	rand.Seed(time.Now().UnixMicro())

	// Generate random instance number
	rand.Seed(time.Now().UnixMicro())
	pid := rand.Int()
	nodeId = fmt.Sprintf("node-%d", pid)

	// Obtain HTML element
	inputEle = js.Global().Get("document").Call("getElementById", "msgrecv")

	// Step 1 - Setup schema tree (supposed to be shared knowledge of all nodes)
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/chat")
	syncNode = &demo.SvsNode{}
	err := tree.PutNode(path, syncNode)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}
	syncNode.Set("ChannelSize", 1000)
	syncNode.Set("SyncInterval", 15*time.Second)
	syncNode.Set("AggregateInterval", 100*time.Millisecond)

	// Step 2 - Setup policies (part is shared by all nodes)
	demo.NewFixedKeySigner([]byte(HmacKey)).Apply(syncNode)
	demo.NewFixedKeyIntSigner([]byte(HmacKey)).Apply(syncNode)
	demo.NewMemStoragePolicy().Apply(syncNode)
	path, _ = enc.NamePatternFromStr("/32=notif")
	demo.NewRegisterPolicy().Apply(syncNode.At(path))
	path, _ = enc.NamePatternFromStr("/<8=nodeId>")
	fmt.Printf("Node ID: %s\n", nodeId)
	demo.NewRegisterPolicy2(enc.Matching{
		"nodeId": []byte(nodeId),
	}).Apply(syncNode.At(path))
	syncNode.Set("SelfNodeId", []byte(nodeId)) // Should belong to step 4

	// Step 3 - Start engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewWasmWsFace("ws", "127.0.0.1:9696", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	err = app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Step 4 & 5 - Configuration and attach schema
	// The configuration part has not been designed now.
	prefix, _ := enc.NameFromStr("/example/schema/chatApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// Set callback and start serving
	inputEle.Call("addEventListener", "beforeinput", js.FuncOf(onBeforeInput))

	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Routine: On data received, modify document
	textDoc = *crdt.NewTextDoc(uint64(pid))
	go func() {
		defer wg.Done()
		ch := syncNode.MissingDataChannel()
		for {
			select {
			case missData := <-ch:
				for i := missData.StartSeq; i < missData.EndSeq; i++ {
					ret, data := (<-syncNode.Need(missData.NodeId, i, enc.Matching{}, schema.Context{})).Get()
					if ret != ndn.InterestResultData {
						fmt.Printf("Data fetching failed for (%s, %d): %+v\n", string(missData.NodeId), i, ret)
					} else {
						dataLock.Lock()
						fmt.Printf("Fetched (%s, %d)\n", string(missData.NodeId), i)
						rec, err := crdt.ParseRecord(enc.NewWireReader(data), false)
						if err != nil {
							log.Errorf("unable to parse record: %+v", err)
							continue
						}
						textDoc.HandleRecord(rec)
						fmt.Printf("Current: %s\n", textDoc.GetText())
						inputEle.Set("value", textDoc.GetText())
						dataLock.Unlock()
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Simply wait forever
	fmt.Printf("Start serving ...\n")
	wg.Wait()
}
