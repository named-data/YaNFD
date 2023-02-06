// This example uses the old schema implemementation and does not work
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

	"github.com/apex/log"
	"github.com/gorilla/websocket"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/demo"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

var homeHtmlTmp []byte
var homeHtml string
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const HmacKey = "Hello, World!"

var app *basic_engine.Engine
var tree *schema.Tree
var syncNode *demo.SvsNode
var wsConn *websocket.Conn // 1 ws connection supported
var msgList []string
var dataLock sync.Mutex
var nodeId string

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

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
			continue
		}
		// print out that message
		dataLock.Lock()
		syncNode.NewData(enc.Wire{p}, schema.Context{})
		msgText := fmt.Sprintf("%s[%d]: %s", string(nodeId), syncNode.MySequence(), p)
		fmt.Printf("received: %s\n", msgText)
		if err := wsConn.WriteMessage(messageType, []byte(msgText)); err != nil {
			wsConn = nil
			running = false
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
		fmt.Println(err)
	}
	// helpful log statement to show connections
	fmt.Println("Client Connected")
	if err != nil {
		fmt.Println(err)
	}

	if wsConn == nil {
		wsConn = ws
	} else {
		ws.Close()
	}
	// Give existing knowledges
	dataLock.Lock()
	for _, msg := range msgList {
		wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
	dataLock.Unlock()
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
		logger.Fatal("Insufficient argument. Please input a port number uniquely used by this instance.")
		return
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		logger.Fatal("Invalid argument")
		return
	}

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

	// Step 1 - Setup schema tree (supposed to be shared knowledge of all nodes)
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/chat")
	syncNode = &demo.SvsNode{}
	err = tree.PutNode(path, syncNode)
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
	nodeId = fmt.Sprintf("node-%d", port)
	fmt.Printf("Node ID: %s\n", nodeId)
	demo.NewRegisterPolicy2(enc.Matching{
		"nodeId": []byte(nodeId),
	}).Apply(syncNode.At(path))
	syncNode.Set("SelfNodeId", []byte(nodeId)) // Should belong to step 4

	// Step 3 - Start engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd.sock", true)
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
	msgList = make([]string, 0)
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
						fmt.Printf("Fetched (%s, %d): %s\n", string(missData.NodeId), i, string(data.Join()))
						msg := fmt.Sprintf("%s[%d]: %s", string(missData.NodeId), i, string(data.Join()))
						msgList = append(msgList, msg)
						if wsConn != nil {
							wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
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
