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
	_ "github.com/zjkmxy/go-ndn/pkg/schema/demosec"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

const SchemaJson = `{
  "nodes": {
    "/randomData/<v=time>": {
      "type": "LeafNode",
      "attrs": {
        "CanBePrefix": false,
        "MustBeFresh": true,
        "Lifetime": 6000,
        "Freshness": 1000,
        "ValidDuration": 3153600000000.0
      }
    },
    "/contentKey": {
      "type": "ContentKeyNode",
      "attrs": {
        "Lifetime": 6000,
        "Freshness": 1000,
        "ValidDuration": 3153600000000.0
      }
    }
  },
  "policies": [
    {
      "type": "RegisterPolicy",
      "path": "/",
      "attrs": {
        "RegisterIf": "$isProducer"
      }
    },
    {
      "type": "Sha256Signer",
      "path": "/randomData/<v=time>",
      "attrs": {}
    },
    {
      "type": "FixedHmacSigner",
      "path": "/contentKey/<contentKeyID>",
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
const HmacKey = "Hello, World!"

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
	tree := schema.CreateFromJson(SchemaJson, map[string]any{
		"$isProducer": false,
		"$hmacKey":    HmacKey,
	})

	// Start engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd/nfd.sock", true)
	app := basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
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
	path, _ := enc.NamePatternFromStr("/randomData/<v=time>")
	node := tree.At(path)
	mNode := node.Apply(enc.Matching{
		"time": enc.Nat(ver).Bytes(),
	})
	result := <-mNode.Call("NeedChan").(chan schema.NeedResult)
	switch result.Status {
	case ndn.InterestResultNack:
		fmt.Printf("Nacked with reason=%d\n", *result.NackReason)
	case ndn.InterestResultTimeout:
		fmt.Printf("Timeout\n")
	case ndn.InterestCancelled:
		fmt.Printf("Canceled\n")
	case ndn.InterestResultData:
		fmt.Printf("Data received\n")
	}
	if result.Status != ndn.InterestResultData {
		return
	}

	path, _ = enc.NamePatternFromStr("/contentKey")
	ckMNode := tree.At(path).Apply(enc.Matching{})
	plainText, ok := ckMNode.Call("Decrypt", result.Content).(enc.Wire)
	if !ok || plainText == nil {
		fmt.Printf("Unable to decrypt data\n")
	} else {
		fmt.Printf("Data: %s\n", string(plainText.Join()))
	}
}
