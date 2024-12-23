package tools

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	enc "github.com/pulsejet/ndnd/std/encoding"
	"github.com/pulsejet/ndnd/std/engine"
	"github.com/pulsejet/ndnd/std/log"
	"github.com/pulsejet/ndnd/std/ndn"
	"github.com/pulsejet/ndnd/std/utils"
)

type PingClient struct {
	args   []string
	prefix enc.Name
	name   enc.Name
	app    ndn.Engine
}

func RunPingClient(args []string) {
	PingClient{args: args}.Run()
}

func (pc PingClient) usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <prefix>\n", pc.args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Ping a NDN name prefix using Interests with name ndn:/name/prefix/ping/number.\n")
	fmt.Fprintf(os.Stderr, "The numbers in the Interests are randomly generated\n")
}

func (pc PingClient) send(seq uint64) {
	name := append(pc.name, enc.NewSequenceNumComponent(seq))

	cfg := &ndn.InterestConfig{
		Lifetime: utils.IdPtr(4 * time.Second),
		Nonce:    utils.ConvertNonce(pc.app.Timer().Nonce()),
	}

	interest, err := pc.app.Spec().MakeInterest(name, cfg, nil, nil)
	if err != nil {
		log.Errorf("Unable to make Interest: %+v", err)
		return
	}

	t1 := time.Now()
	err = pc.app.Express(interest, func(args ndn.ExpressCallbackArgs) {
		t2 := time.Now()

		switch args.Result {
		case ndn.InterestResultNack:
			fmt.Printf("nack from %s: seq=%d with reason=%d\n", pc.prefix, seq, args.NackReason)
		case ndn.InterestResultTimeout:
			fmt.Printf("timeout from %s: seq=%d\n", pc.prefix, seq)
		case ndn.InterestCancelled:
			fmt.Printf("canceled from %s: seq=%d\n", pc.prefix, seq)
		case ndn.InterestResultData:
			ms := float64(t2.Sub(t1).Microseconds()) / 1000.0
			fmt.Printf("content from %s: seq=%d, time=%f ms\n", pc.prefix, seq, ms)
		}
	})
	if err != nil {
		log.Errorf("Unable to send Interest: %+v", err)
	}
}

func (pc PingClient) Run() {
	log.SetLevel(log.InfoLevel)

	if len(pc.args) < 2 {
		pc.usage()
		os.Exit(3)
	}

	prefix, err := enc.NameFromStr(pc.args[1])
	if err != nil {
		log.Fatalf("Invalid prefix: %s", pc.args[1])
	}
	pc.prefix = prefix
	pc.name = append(prefix, enc.NewStringComponent(enc.TypeGenericNameComponent, "ping"))

	face := engine.NewUnixFace("/var/run/nfd/nfd.sock")
	pc.app = engine.NewBasicEngine(face)
	err = pc.app.Start()
	if err != nil {
		log.Fatalf("Unable to start engine: %+v", err)
		return
	}
	defer pc.app.Stop()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Second)
	seq := uint64(0)

	fmt.Printf("PING %s\n", pc.name)
	pc.send(seq)

	for {
		select {
		case <-ticker.C:
			seq++
			pc.send(seq)
		case <-sigchan:
			fmt.Println("Interrupted")
			return
		}
	}
}
