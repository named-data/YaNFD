/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	server "github.com/named-data/YaNFD/cmd/yanfdui/server"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/executor"
)

// Version of YaNFD.
var Version string

// Addr of HTTP server.
var Addr string = "localhost:5010"

// HttpBaseDir is the base directory of server files
var HttpBaseDir string = "."

func main() {
	// Parse command line options
	var shouldPrintVersion bool
	flag.BoolVar(&shouldPrintVersion, "version", false, "Print version and exit")
	var configFileName string
	flag.StringVar(&configFileName, "config", "/usr/local/etc/ndn/yanfd.toml", "Configuration file location")
	var memoryBallastSize int
	flag.IntVar(&memoryBallastSize, "memory-ballast", 0, "Enable memory ballast of specified size (in GB) to avoid frequent garbage collection")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version " + Version)
		fmt.Println("Copyright (C) 2020-2021 Eric Newberry")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	if runtime.GOOS == "windows" && configFileName[0] == '/' {
		configFileName = os.ExpandEnv("${APPDATA}\\ndn\\yanfd.toml")
	}

	config := executor.YaNFDConfig{
		Version:           Version,
		ConfigFileName:    configFileName,
		DisableEthernet:   false,
		DisableUnix:       false,
		LogFile:           "",
		CpuProfile:        "",
		MemProfile:        "",
		BlockProfile:      "",
		MemoryBallastSize: memoryBallastSize,
	}

	yanfd := executor.NewYaNFD(&config)
	yanfd.Start()

	// Start HTTP server
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	srv := server.StartHttpServer(httpServerExitDone, Addr, HttpBaseDir, configFileName)
	server.OpenBrowser("http://" + Addr)

	// Set up signal handler channel and wait for interrupt
	core.LogInfo("Main", "HTTP server started, serving at http://", Addr)
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	core.LogInfo("Main", "Received signal ", receivedSig, " - exiting")

	// Stop HTTP server
	if err := srv.Shutdown(context.Background()); err != nil {
		core.LogInfo("Main", "Error in shutting down HTTP server: ", err)
	}
	yanfd.Stop()
	httpServerExitDone.Wait()
}
