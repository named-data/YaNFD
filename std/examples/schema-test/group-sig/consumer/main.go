package main

import (
	"fmt"
	"os"
	"strconv"

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
	log.SetLevel(log.InfoLevel)
	logger := log.WithField("module", "main")

	if len(os.Args) < 2 {
		logger.Fatal("Insufficient argument. Please input the version number given by the producer.")
		return
	}
	ver, err := strconv.Atoi(os.Args[1])
	if err != nil {
		logger.Fatal("Invalid argument")
		return
	}
	// Setup schema tree
	tree = &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/lorem/<v=time>")
	node := &demo.GroupSigNode{}
	err = tree.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}
	node.Set("Threshold", 80)

	// Setup policies
	demo.NewFixedKeySigner([]byte(HmacKey)).Apply(node) // Only affect the metadata node
	demo.NewMemStoragePolicy().Apply(node)

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
	prefix, _ := enc.NameFromStr("/example/schema/groupSigApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// Fetch the data
	context := schema.Context{}
	result, content := (<-node.Need(enc.Matching{
		"time": uint64(ver),
	}, context)).Get()
	switch result {
	case ndn.InterestResultNone:
		fmt.Printf("Fetching failed. Please see log for detailed reason.\n")
	case ndn.InterestResultNack:
		fmt.Printf("Nacked with reason=%d\n", context[schema.CkNackReason])
	case ndn.InterestResultTimeout:
		fmt.Printf("Timeout\n")
	case ndn.InterestCancelled:
		fmt.Printf("Canceled\n")
	case ndn.InterestResultData:
		fmt.Printf("Received Data: \n")
		fmt.Printf("%s", string(content.Join()))
	}
}
