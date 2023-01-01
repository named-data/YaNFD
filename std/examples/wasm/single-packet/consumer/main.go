package main

import (
	"fmt"
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

func main() {
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")

	// Setup schema tree
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/randomData/<t=time>")
	node := &schema.ExpressPoint{}
	err := tree.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to set the node: %+v", err)
		return
	}
	node.Set(schema.PropCanBePrefix, false)
	node.Set(schema.PropMustBeFresh, true)
	node.Set(schema.PropLifetime, 6*time.Second)
	passAllChecker := func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, schema.Context) schema.ValidRes {
		return schema.VrPass
	}
	node.Get(schema.PropOnValidateData).(*schema.Event[*schema.NodeValidateEvent]).Add(&passAllChecker)

	// Setup engine
	timer := basic_engine.NewTimer()
	// face := basic_engine.NewStreamFace("unix", "/var/run/nfd.sock", true)
	face := basic_engine.NewWasmWsFace("ws", "127.0.0.1:9696", true)
	app = basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	err = app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Attach the schema
	prefix, _ := enc.NameFromStr("/example/testApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// Fetch the data
	context := schema.Context{}
	result, content := (<-node.Need(enc.Matching{
		"time": utils.MakeTimestamp(timer.Now()),
	}, nil, nil, context)).Get()
	switch result {
	case ndn.InterestResultNack:
		fmt.Printf("Nacked with reason=%d\n", context[schema.CkNackReason])
	case ndn.InterestResultTimeout:
		fmt.Printf("Timeout\n")
	case ndn.InterestCancelled:
		fmt.Printf("Canceled\n")
	case ndn.InterestResultData:
		fmt.Printf("Received Data: %+v\n", string(content.Join()))
	}
}
