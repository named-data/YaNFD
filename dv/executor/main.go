package executor

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/goccy/go-yaml"

	"github.com/pulsejet/ndnd/std/log"
)

func Main(args []string) {
	var cfgFile string = "/etc/ndn/dv.yml"
	if len(args) >= 2 {
		cfgFile = args[1]
	}

	cfgBytes, err := os.ReadFile(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read configuration file: %s\n", err)
		os.Exit(3)
	}

	dc := DefaultConfig()
	if err = yaml.Unmarshal(cfgBytes, &dc); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse configuration file: %s\n", err)
		os.Exit(3)
	}

	log.SetLevel(log.InfoLevel)

	dve, err := NewDvExecutor(dc)
	if err != nil {
		panic(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	quitchan := make(chan bool, 1)
	go func() {
		if err = dve.Start(); err != nil {
			panic(err)
		}
		quitchan <- true
	}()

	for {
		select {
		case <-sigchan:
			dve.Stop()
		case <-quitchan:
			return
		}
	}
}
