package main

import (
	"os"
	"time"

	enc "github.com/pulsejet/ndnd/std/encoding"
	"github.com/pulsejet/ndnd/std/engine"
	"github.com/pulsejet/ndnd/std/log"
	"github.com/pulsejet/ndnd/std/object"
)

func main() {
	log.SetLevel(log.InfoLevel)

	if len(os.Args) < 2 {
		log.Fatalf("Usage: catchunks <name>")
	}

	// get name from cli
	name, err := enc.NameFromStr(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid name: %s", os.Args[1])
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

	done := make(chan *object.ConsumeState)
	t1, t2 := time.Now(), time.Now()
	byteCount := 0

	// calling Content() on a status object clears the buffer
	// and returns the new data the next time it is called
	write := func(status *object.ConsumeState) {
		content := status.Content()
		os.Stdout.Write(content)
		byteCount += len(content)
	}

	// fetch object
	cli.Consume(name, func(status *object.ConsumeState) bool {
		if status.IsComplete() {
			t2 = time.Now()
			write(status)
			done <- status
		}

		if status.Progress()%1000 == 0 {
			log.Debugf("Progress: %.2f%%", float64(status.Progress())/float64(status.ProgressMax())*100)
			write(status)
		}

		return true
	})
	state := <-done

	if state.Error() != nil {
		log.Errorf("Error fetching object: %+v", state.Error())
		return
	}

	// statistics
	log.Infof("Object fetched: %s", state.Name())
	log.Infof("Content: %d bytes", byteCount)
	log.Infof("Time taken: %s", t2.Sub(t1))
	log.Infof("Throughput: %f Mbit/s", float64(byteCount*8)/t2.Sub(t1).Seconds()/1e6)
}
