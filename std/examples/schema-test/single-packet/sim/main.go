// TODO: This does not work. Commit just to record what I did.
package main

import (
	"fmt"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/engine/sim"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/schema/demo"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

func passAll(enc.Name, enc.Wire, ndn.Signature) bool {
	return true
}

func main() {
	log.SetLevel(log.DebugLevel)
	logger := log.WithField("module", "main")

	var app1, app2 *basic_engine.Engine

	// Setup nodes
	timer := sim.NewTimer()
	link := sim.NewSharedMulticast(timer, 2, 0.0, 20*time.Millisecond)

	// Setup schema trees
	// 1 is the producer, 2 is the consumer
	tree1 := &schema.Tree{}
	path, _ := enc.NamePatternFromStr("/randomData/<t=time>")
	node := &schema.ExpressPoint{}
	err := tree1.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to setup node: %+v", err)
		return
	}
	node.Set(schema.PropCanBePrefix, false)
	node.Set(schema.PropMustBeFresh, true)
	node.Set(schema.PropLifetime, 6*time.Second)
	onInterest := func(matching enc.Matching, appParam enc.Wire, reply ndn.ReplyFunc, context schema.Context) bool {
		fmt.Printf(">> I: timestamp: %d\n", matching["time"].(uint64))
		content := []byte("Hello, world!")
		name := context[schema.CkName].(enc.Name)
		wire, _, err := app1.Spec().MakeData(
			name,
			&ndn.DataConfig{
				ContentType: utils.IdPtr(ndn.ContentTypeBlob),
				Freshness:   utils.IdPtr(10 * time.Second),
			},
			enc.Wire{content},
			sec.NewSha256Signer())
		if err != nil {
			log.WithField("module", "main").Errorf("unable to encode data: %+v", err)
			return true
		}
		err = reply(wire)
		if err != nil {
			log.WithField("module", "main").Errorf("unable to reply with data: %+v", err)
			return true
		}
		fmt.Printf("<< D: %s\n", name.String())
		fmt.Printf("ontent: (size: %d)\n", len(content))
		fmt.Printf("\n")
		return true
	}
	schema.AddEventListener(node, schema.PropOnInterest, onInterest)

	tree2 := &schema.Tree{}
	path, _ = enc.NamePatternFromStr("/randomData/<t=time>")
	node = &schema.ExpressPoint{}
	err = tree2.PutNode(path, node)
	if err != nil {
		logger.Fatalf("Unable to setup node: %+v", err)
		return
	}
	node.Set(schema.PropCanBePrefix, false)
	node.Set(schema.PropMustBeFresh, true)
	node.Set(schema.PropLifetime, 6*time.Second)
	passAllChecker := func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, schema.Context) schema.ValidRes {
		return schema.VrPass
	}
	node.Get(schema.PropOnValidateData).(*schema.Event[*schema.NodeValidateEvent]).Add(&passAllChecker)

	// Setup policies
	// The prefix registered is at the root
	demo.NewRegisterPolicy().Apply(tree1.Root)

	// Start engines
	app1 = basic_engine.NewEngine(link.Face(0), timer, sec.NewSha256IntSigner(timer), passAll)
	err = app1.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}
	app2 = basic_engine.NewEngine(link.Face(1), timer, sec.NewSha256IntSigner(timer), passAll)
	err = app2.Start()
	if err != nil {
		logger.Fatalf("Unable to start engine: %+v", err)
		return
	}

	// Attach schema
	// Go routine is needed as tree.attach needs timer moving forward to receive the registration result
	go func() {
		prefix, _ := enc.NameFromStr("/example/testApp")
		err = tree1.Attach(prefix, app1)
		if err != nil {
			logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
			return
		}
		err = tree2.Attach(prefix, app2)
		if err != nil {
			logger.Fatalf("Unable to attach the schema to the engine: %+v", err)
			return
		}
	}()

	for i := 0; i < 20; i++ {
		// First warm up the engine, so we are sure that the rest is executed after attach
		time.Sleep(10 * time.Millisecond) // leave time for attach to execute
		// TODO: Run for may be better
		timer.RunUntil(timer.Now().Add(10 * time.Second))
	}

	// Send Interest
	timer.Schedule(1*time.Second, func() {
		path, _ := enc.NamePatternFromStr("/randomData/<t=time>")
		node := tree1.At(path).(*schema.ExpressPoint)
		fmt.Printf("Express Int\n")
		go func() {
			result, content := node.Need(enc.Matching{
				"time": utils.MakeTimestamp(timer.Now()),
			}, nil, nil, schema.Context{})
			switch result {
			case ndn.InterestResultNack:
				fmt.Printf("Nacked\n")
			case ndn.InterestResultTimeout:
				fmt.Printf("Timeout\n")
			case ndn.InterestCancelled:
				fmt.Printf("Canceled\n")
			case ndn.InterestResultData:
				fmt.Printf("Received Data: %+v\n", content.Join())
			}
		}()
	})

	for i := 0; i < 20; i++ {
		time.Sleep(10 * time.Millisecond)
		timer.RunUntil(timer.Now().Add(10 * time.Second))
	}
}
