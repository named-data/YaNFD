package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zjkmxy/go-ndn/examples/schema-test/shared-doc/crdt"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/engine"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/svs"
)

var homeHtmlTmp []byte
var homeHtml string
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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
var wsConn *websocket.Conn // 1 ws connection supported
var textDoc crdt.TextDoc
var dataLock sync.Mutex
var nodeId string

func homePage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(homeHtml))
}

func wsReader() {
	running := true
	for running {
		// read in a message
		messageType, p, err := wsConn.ReadMessage()
		if err != nil {
			dataLock.Lock()
			wsConn = nil
			dataLock.Unlock()
			running = false
			break
		}
		// print out that message
		eventText := string(p)
		fmt.Printf("Event: %s\n", eventText)
		args := strings.Split(eventText, ",")
		validReq := true
		dataLock.Lock()
		if len(args) == 3 && (args[2] == "insertText" || args[2] == "deleteContentBackward") {
			pos, _ := strconv.ParseInt(args[0], 10, 32)
			var rec *crdt.Record // the new record to send
			if args[2] == "insertText" {
				rec = textDoc.Insert(int(pos), args[1])
			} else {
				rec = textDoc.Delete(int(pos))
			}
			if rec != nil {
				syncNode.Call("NewData", rec.Encode())
			} else {
				validReq = false
			}
		} else {
			// Invalid request
			validReq = false
		}
		if !validReq {
			cur := textDoc.GetText()
			if err := wsConn.WriteMessage(messageType, []byte(cur)); err != nil {
				wsConn = nil
				running = false
			}
		}
		dataLock.Unlock()
	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("unable to crate ws endpoint: %+v", err)
	}
	// helpful log statement to show connections
	fmt.Println("Client Connected")

	if wsConn == nil {
		wsConn = ws
	} else {
		ws.Close()
	}
	// Give existing knowledges
	dataLock.Lock()
	msg := textDoc.GetText()
	dataLock.Unlock()
	wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
	// Fetch messages sent by the client
	wsReader()
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	// Note: remember to ` nfdc strategy set /example/schema /localhost/nfd/strategy/multicast `
	log.SetLevel(log.ErrorLevel)
	logger := log.WithField("module", "main")
	rand.Seed(time.Now().UnixMicro())

	// Parse port number
	if len(os.Args) < 2 {
		logger.Fatal("Insufficient argument. Please input a port number.")
		return
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		logger.Fatal("Invalid argument")
		return
	}
	nodeId = fmt.Sprintf("node-%d", port)

	// Load HTML UI file to serve
	file, err := os.Open("home.html")
	if err != nil {
		logger.Fatalf("Failed to open home.html: %+v", err)
	}
	homeHtmlTmp, err = io.ReadAll(file)
	if err != nil {
		logger.Fatalf("Failed to read home.html: %+v", err)
	}
	file.Close()
	temp, err := template.New("HTML").Parse(string(homeHtmlTmp))
	if err != nil {
		logger.Fatalf("Failed to create template: %+v", err)
	}
	strBuilder := strings.Builder{}
	temp.Execute(&strBuilder, port)
	homeHtml = strBuilder.String()

	// Setup schema tree
	tree := schema.CreateFromJson(SchemaJson, map[string]any{
		"$hmacKey": HmacKey,
		"$nodeId":  nodeId,
	})

	// Start engine
	face := engine.NewUnixFace("/var/run/nfd/nfd.sock")
	app := engine.NewBasicEngine(face)
	err = app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Stop()

	// Attach schema
	prefix, _ := enc.NameFromStr("/example/schema/sharedDoc")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()
	syncNode = tree.At(enc.NamePattern{}).Apply(enc.Matching{})

	// Start serving HTTP routes
	setupRoutes()
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: nil}

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(2)
	// Routine 1: HTTP and WS server
	go func() {
		defer wg.Done()
		server.ListenAndServe()
	}()

	// Routine 2: On data received, send over ws
	textDoc = *crdt.NewTextDoc(uint64(port)) // Use port number as producer ID
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
						if wsConn != nil {
							wsConn.WriteMessage(websocket.TextMessage, []byte(textDoc.GetText()))
						}
						dataLock.Unlock()
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for keyboard quit signal
	sigChannel := make(chan os.Signal, 1)
	fmt.Printf("Start serving on   http://localhost:%d/   ...\n", port)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	logger.Infof("Received signal %+v - exiting\n", receivedSig)
	cancel()
	server.Shutdown(context.Background())
	wg.Wait()
}
