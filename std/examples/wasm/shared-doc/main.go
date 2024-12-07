//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"syscall/js"
	"time"

	"github.com/zjkmxy/go-ndn/examples/schema-test/shared-doc/crdt"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/svs"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

const HmacKey = "Hello, World!"

const SchemaJson = `{
  "nodes": {
    "/": {
      "type": "SvsNode",
      "attrs": {
        "ChannelSize": 1000,
        "SyncInterval": 15000,
        "SuppressionInterval": 100,
        "SelfNodeId": "$nodeId",
        "BaseMatching": {}
      }
    }
  },
  "policies": [
    {
      "type": "RegisterPolicy",
      "path": "/32=notif",
      "attrs": {}
    },
    {
      "type": "RegisterPolicy",
      "path": "/<8=nodeId>",
      "attrs": {
        "Patterns": {
          "nodeId": "$nodeId"
        }
      }
    },
    {
      "type": "FixedHmacSigner",
      "path": "/<8=nodeId>/<seq=seqNo>",
      "attrs": {
        "KeyValue": "$hmacKey"
      }
    },
    {
      "type": "FixedHmacIntSigner",
      "path": "/32=notif",
      "attrs": {
        "KeyValue": "$hmacKey"
      }
    },
    {
      "type": "MemStorage",
      "path": "/",
      "attrs": {}
    }
  ]
}`

var syncNode *schema.MatchedNode
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
			syncNode.Call("NewData", rec.Encode())
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

	// Setup schema tree
	tree := schema.CreateFromJson(SchemaJson, map[string]any{
		"$hmacKey": HmacKey,
		"$nodeId":  nodeId,
	})

	// Start engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewWasmWsFace("ws", "127.0.0.1:9696", true)
	app := basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	err := app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Configuration and attach schema
	// The configuration part has not been designed now.
	prefix, _ := enc.NameFromStr("/example/schema/sharedDocWasm")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()
	syncNode = tree.At(enc.NamePattern{}).Apply(enc.Matching{})

	// Set callback and start serving
	inputEle.Call("addEventListener", "beforeinput", js.FuncOf(onBeforeInput))

	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Routine: On data received, modify document
	textDoc = *crdt.NewTextDoc(uint64(pid))
	go func() {
		defer wg.Done()
		ch := syncNode.Call("MissingDataChannel").(chan svs.MissingData)
		for {
			select {
			case missData := <-ch:
				for i := missData.StartSeq; i < missData.EndSeq; i++ {
					dataName := syncNode.Call("GetDataName", missData.NodeId, i).(enc.Name)
					mLeafNode := tree.Match(dataName)
					result := <-mLeafNode.Call("NeedChan").(chan schema.NeedResult)
					if result.Status != ndn.InterestResultData {
						fmt.Printf("Data fetching failed for (%s, %d): %+v\n", string(missData.NodeId), i, result.Status)
					} else {
						dataLock.Lock()
						fmt.Printf("Fetched (%s, %d)\n", string(missData.NodeId), i)
						rec, err := crdt.ParseRecord(enc.NewWireReader(result.Content), false)
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
