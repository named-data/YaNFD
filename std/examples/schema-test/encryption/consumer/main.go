package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/demo"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

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
	path, _ := enc.NamePatternFromStr("/randomData/<v=time>")
	node := &schema.LeafNode{}
	err = tree.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}
	node.Set(schema.PropCanBePrefix, false)
	node.Set(schema.PropMustBeFresh, true)
	node.Set(schema.PropLifetime, 6*time.Second)
	node.Set(schema.PropFreshness, 1*time.Second)
	node.Set(schema.PropValidDuration, 876000*time.Hour)
	node.Set(schema.PropDataSigner, sec.NewSha256Signer())
	passAllChecker := func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, schema.Context) schema.ValidRes {
		return schema.VrPass
	}
	node.Get(schema.PropOnValidateData).(*schema.Event[*schema.NodeValidateEvent]).Add(&passAllChecker)
	path, _ = enc.NamePatternFromStr("/contentKey")
	ckNode := &demo.ContentKeyNode{}
	err = tree.PutNode(path, ckNode)
	if err != nil {
		logger.Fatalf("Unable to construst the schema tree: %+v", err)
		return
	}

	// Setup policies
	memStorage := demo.NewMemStoragePolicy()
	memStorage.Apply(node)
	memStorage.Apply(ckNode)

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
	prefix, _ := enc.NameFromStr("/example/schema/encryptionApp")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// Fetch the data
	context := schema.Context{}
	matching := enc.Matching{"time": uint64(ver)}
	result, content := (<-node.Need(matching, nil, nil, context)).Get()
	switch result {
	case ndn.InterestResultNack:
		fmt.Printf("Nacked with reason=%d\n", context[schema.CkNackReason])
	case ndn.InterestResultTimeout:
		fmt.Printf("Timeout\n")
	case ndn.InterestCancelled:
		fmt.Printf("Canceled\n")
	case ndn.InterestResultData:
		plainText, err := ckNode.Decrypt(matching, content)
		if err != nil {
			logger.Fatalf("Unable to encrypt data: %+v", err)
			return
		}
		fmt.Printf("Received Data: %+v\n", plainText.Join())
	}
}
