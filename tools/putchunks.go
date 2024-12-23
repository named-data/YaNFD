package tools

import (
	"fmt"
	"os"

	enc "github.com/pulsejet/ndnd/std/encoding"
	"github.com/pulsejet/ndnd/std/engine"
	"github.com/pulsejet/ndnd/std/log"
	"github.com/pulsejet/ndnd/std/object"
)

func putChunksUsage(args []string) {
	fmt.Fprintf(os.Stderr, "Usage: %s <name>\n", args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Publish data under the specified prefix.\n")
	fmt.Fprintf(os.Stderr, "This tool expects data from the standard input.\n")
}

func PutChunks(args []string) {
	log.SetLevel(log.InfoLevel)

	if len(args) < 2 {
		putChunksUsage(args)
		os.Exit(3)
	}

	// get name from cli
	name, err := enc.NameFromStr(args[1])
	if err != nil {
		log.Fatalf("Invalid name: %s", args[1])
	}

	// start face and engine
	face := engine.NewUnixFace("/var/run/nfd/nfd.sock")
	engine := engine.NewBasicEngine(face)
	err = engine.Start()
	if err != nil {
		log.Errorf("Unable to start engine: %+v", err)
		return
	}
	defer engine.Stop()

	// start object client
	cli := object.NewClient(engine, object.NewMemoryStore())
	err = cli.Start()
	if err != nil {
		log.Errorf("Unable to start object client: %+v", err)
		return
	}
	defer cli.Stop()

	// read from stdin till eof
	var content enc.Wire
	for {
		buf := make([]byte, 1e6) // 1MB
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}
		content = append(content, buf[:n])
	}

	// produce object
	vname, err := cli.Produce(object.ProduceArgs{
		Name:    name,
		Content: content,
	})
	if err != nil {
		log.Fatalf("Unable to produce object: %+v", err)
		return
	}

	content = nil // gc
	log.Infof("Object produced: %s", vname)

	// register route to the object
	err = engine.RegisterRoute(name)
	if err != nil {
		log.Fatalf("Unable to register route: %+v", err)
		return
	}

	// wait forever
	// TODO: quit on SIGTERM, SIGINT or face failure
	select {}
}
