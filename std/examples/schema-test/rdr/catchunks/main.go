package main

import (
	"fmt"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	_ "github.com/zjkmxy/go-ndn/pkg/schema/rdr"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

const SchemaJson = `{
  "nodes": {
    "/": {
      "type": "RdrNode",
      "attrs": {
        "MetaFreshness": 10,
        "MaxRetriesForMeta": 2,
        "MetaLifetime": 6000,
        "Lifetime": 6000,
        "Freshness": 3153600000000,
        "ValidDuration": 3153600000000,
        "SegmentSize": 80,
        "MaxRetriesOnFailure": 3,
        "Pipeline": "SinglePacket"
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
      "path": "/32=metadata/<v=versionNumber>/seg=0"
    },
    {
      "type": "Sha256Signer",
      "path": "/32=metadata"
    },
    {
      "type": "Sha256Signer",
      "path": "/<v=versionNumber>/<seg=segmentNumber>"
    },
    {
      "type": "MemStorage",
      "path": "/",
      "attrs": {}
    }
  ]
}`

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	log.SetLevel(log.DebugLevel)
	logger := log.WithField("module", "main")

	// Setup schema tree
	tree := schema.CreateFromJson(SchemaJson, map[string]any{
		"$isProducer": false,
	})

	// Setup engine
	timer := basic_engine.NewTimer()
	face := basic_engine.NewStreamFace("unix", "/var/run/nfd/nfd.sock", true)
	app := basic_engine.NewEngine(face, timer, sec.NewSha256IntSigner(timer), passAll)
	err := app.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer app.Shutdown()

	// Attach the schema
	prefix, _ := enc.NameFromStr("/example/schema/rdr")
	err = tree.Attach(prefix, app)
	if err != nil {
		logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
		return
	}
	defer tree.Detach()

	// Fetch the data
	mNode := tree.Root().Apply(enc.Matching{})
	result := <-mNode.Call("NeedChan").(chan schema.NeedResult)
	switch result.Status {
	case ndn.InterestResultNack:
		fmt.Printf("Nacked with reason=%d\n", *result.NackReason)
	case ndn.InterestResultTimeout:
		fmt.Printf("Timeout\n")
	case ndn.InterestCancelled:
		fmt.Printf("Canceled\n")
	case ndn.InterestResultData:
		fmt.Printf("Received Data: %+v\n", string(result.Content.Join()))
	}
}
