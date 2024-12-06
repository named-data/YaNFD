/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/executor"
)

// Version of YaNFD.
var Version string

func main() {
	// Parse command line options
	var shouldPrintVersion bool
	flag.BoolVar(&shouldPrintVersion, "version", false, "Print version and exit")
	var configFileName string
	flag.StringVar(&configFileName, "config", "/usr/local/etc/ndn/yanfd.toml", "Configuration file location")
	var disableUnix bool
	flag.BoolVar(&disableUnix, "disable-unix", false,
		"Disable Unix stream transports (deprecated; set.faces.unix.enabled=false in config file instead)")
	var cpuProfile string
	flag.StringVar(&cpuProfile, "cpu-profile", "", "Enable CPU profiling (output to specified file)")
	var memProfile string
	flag.StringVar(&memProfile, "mem-profile", "", "Enable memory profiling (output to specified file)")
	var blockProfile string
	flag.StringVar(&blockProfile, "block-profile", "", "Enable block profiling (output to specified file)")
	var memoryBallastSize int
	flag.IntVar(&memoryBallastSize, "memory-ballast", 0,
		"Enable memory ballast of specified size (in GB) to avoid frequent garbage collection")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version " + Version)
		fmt.Println("Copyright (C) 2020-2021 Eric Newberry")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	config := executor.YaNFDConfig{
		Version:           Version,
		ConfigFileName:    configFileName,
		DisableUnix:       disableUnix,
		LogFile:           "",
		CpuProfile:        cpuProfile,
		MemProfile:        memProfile,
		BlockProfile:      blockProfile,
		MemoryBallastSize: memoryBallastSize,
	}

	yanfd := executor.NewYaNFD(&config)
	yanfd.Start()

	// Set up signal handler channel and wait for interrupt
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	core.LogInfo("Main", "Received signal ", receivedSig, " - exiting")

	yanfd.Stop()
}
