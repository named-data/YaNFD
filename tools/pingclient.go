package tools

import (
	"fmt"
	"math/rand/v2"
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

	nRecv    int
	nNack    int
	nTimeout int

	totalTime  time.Duration
	totalCount int

	rttMin time.Duration
	rttMax time.Duration
	rttAvg time.Duration
}

func RunPingClient(args []string) {
	(&PingClient{args: args}).run()
}

func (pc *PingClient) usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <prefix>\n", pc.args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Ping a NDN name prefix using Interests with name ndn:/name/prefix/ping/number.\n")
	fmt.Fprintf(os.Stderr, "The numbers in the Interests are randomly generated\n")
}

func (pc *PingClient) send(seq uint64) {
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

	pc.totalCount++
	t1 := time.Now()

	err = pc.app.Express(interest, func(args ndn.ExpressCallbackArgs) {
		t2 := time.Now()

		switch args.Result {
		case ndn.InterestResultNack:
			fmt.Printf("nack from %s: seq=%d with reason=%d\n", pc.prefix, seq, args.NackReason)
			pc.nNack++
		case ndn.InterestResultTimeout:
			fmt.Printf("timeout from %s: seq=%d\n", pc.prefix, seq)
			pc.nTimeout++
		case ndn.InterestCancelled:
			fmt.Printf("canceled from %s: seq=%d\n", pc.prefix, seq)
			pc.nNack++
		case ndn.InterestResultData:
			fmt.Printf("content from %s: seq=%d, time=%f ms\n",
				pc.prefix, seq,
				float64(t2.Sub(t1).Microseconds())/1000.0)

			pc.nRecv++
			pc.totalTime += t2.Sub(t1)
			pc.rttMin = min(pc.rttMin, t2.Sub(t1))
			pc.rttMax = max(pc.rttMax, t2.Sub(t1))
			pc.rttAvg = pc.totalTime / time.Duration(pc.nRecv)
		}
	})
	if err != nil {
		log.Errorf("Unable to send Interest: %+v", err)
	}
}

func (pc *PingClient) stats() {
	if pc.totalCount == 0 {
		fmt.Printf("No interests transmitted\n")
		return
	}

	fmt.Printf("\n--- %s ping statistics ---\n", pc.prefix)
	fmt.Printf("%d interests transmitted, %d received, %d%% lost\n",
		pc.totalCount, pc.nRecv, (pc.nNack+pc.nTimeout)*100/pc.totalCount)
	fmt.Printf("rtt min/avg/max = %f/%f/%f ms\n",
		float64(pc.rttMin.Microseconds())/1000.0,
		float64(pc.rttAvg.Microseconds())/1000.0,
		float64(pc.rttMax.Microseconds())/1000.0)
}

func (pc *PingClient) run() {
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
	seq := rand.Uint64()

	fmt.Printf("PING %s\n", pc.name)
	pc.send(seq)

	for {
		select {
		case <-ticker.C:
			seq++
			pc.send(seq)
		case <-sigchan:
			pc.stats()
			return
		}
	}
}
