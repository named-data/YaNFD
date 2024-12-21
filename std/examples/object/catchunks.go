package main

import (
	"os"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/engine"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/object"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: catchunks <name>")
	}

	// get name from cli
	name, err := enc.NameFromStr(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid name: %s", os.Args[1])
	}

	log.SetLevel(log.DebugLevel)

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
	cli := object.NewClient(engine)
	err = cli.Start()
	if err != nil {
		log.Errorf("Unable to start object client: %+v", err)
		return
	}
	defer cli.Stop()

	// fetch object
	ch := make(chan *object.ConsumeState)
	t1, t2 := time.Now(), time.Now()
	cli.Consume(name, func(status *object.ConsumeState) bool {
		if status.IsComplete() {
			t2 = time.Now()
			ch <- status
		}

		if status.Progress()%1000 == 0 {
			log.Debugf("Progress: %.2f%%", float64(status.Progress())/float64(status.ProgressMax())*100)
		}

		return true
	})
	log.Debugf("Waiting for object")
	state := <-ch

	if state.Error() != nil {
		log.Errorf("Error fetching object: %+v", state.Error())
		return
	}

	// state.Content() can be called exactly once
	content := state.Content()

	// statistics
	log.Infof("Object fetched: %s", state.Name())
	log.Infof("Content: %d bytes", len(content))
	log.Infof("Time taken: %s", t2.Sub(t1))
	log.Infof("Throughput: %f Mbit/s", float64(len(content)*8)/t2.Sub(t1).Seconds()/1e6)
}
