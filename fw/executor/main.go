package executor

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/goccy/go-yaml"
	"github.com/pulsejet/ndnd/fw/core"
)

var Version string

func Main(args []string) {
	config := &YaNFDConfig{}

	flagset := flag.NewFlagSet("yanfd", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Printf("Usage: %s [configfile] [options]\n", args[0])
		flagset.PrintDefaults()
	}

	var printVersion bool
	flagset.BoolVar(&printVersion, "version", false, "Print version and exit")

	flagset.StringVar(&config.CpuProfile, "cpu-profile", "", "Enable CPU profiling (output to specified file)")
	flagset.StringVar(&config.MemProfile, "mem-profile", "", "Enable memory profiling (output to specified file)")
	flagset.StringVar(&config.BlockProfile, "block-profile", "", "Enable block profiling (output to specified file)")
	flagset.IntVar(&config.MemoryBallastSize, "memory-ballast", 0, "Enable memory ballast of specified size (in GB) to avoid frequent garbage collection")

	flagset.Parse(args[1:])

	if printVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version: ", Version)
		fmt.Println("Copyright (C) 2020-2024 University of California")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	configfile := flagset.Arg(0)
	if configfile == "help" {
		flagset.Usage()
		return
	} else if configfile == "" {
		configfile = "/usr/local/etc/ndn/yanfd.yml"
	}
	config.BaseDir = filepath.Dir(configfile)

	f, err := os.Open(configfile)
	if err != nil {
		panic(errors.New("Unable to open configuration file: " + err.Error()))
	}
	defer f.Close()

	config.Config = core.DefaultConfig()
	dec := yaml.NewDecoder(f, yaml.Strict())
	if err = dec.Decode(config.Config); err != nil {
		panic(errors.New("Unable to parse configuration file: " + err.Error()))
	}

	// create YaNFD instance
	yanfd := NewYaNFD(config)
	yanfd.Start()

	// set up signal handler channel and wait for interrupt
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	core.LogInfo("Main", "Received signal ", receivedSig, " - exiting")

	yanfd.Stop()
}
