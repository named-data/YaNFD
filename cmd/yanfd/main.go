/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face"
	"github.com/eric135/YaNFD/fw"
)

func main() {
	// Parse command line options
	var shouldPrintVersion bool
	flag.BoolVar(&shouldPrintVersion, "version", false, "Print version and exit")
	flag.BoolVar(&shouldPrintVersion, "V", false, "Print version and exit (short)")
	var numForwardingThreads int
	flag.IntVar(&numForwardingThreads, "threads", 8, "Number of forwarding threads")
	flag.IntVar(&numForwardingThreads, "t", 8, "Number of forwarding threads")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version", core.Version)
		fmt.Println("Copyright (C) 2020 Eric Newberry")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	if numForwardingThreads < 1 || numForwardingThreads > fw.MaxFwThreads {
		fmt.Println("Number of forwarding threads must be in range [1,", fw.MaxFwThreads, "]")
		fmt.Println()
		flag.PrintDefaults()
		os.Exit(1)
	}

	core.LogInfo("Main", "Starting NFD")

	// Start management thread
	// TODO

	// Create forwarding threads
	fw.Threads = make(map[int]*fw.Thread)
	for i := 0; i < numForwardingThreads; i++ {
		newThread := fw.NewThread(i)
		fw.Threads[i] = &newThread
		go fw.Threads[i].Run()
	}

	// Initialize face system
	face.FaceTable = face.MakeTable()
	// Create null face
	face.FaceTable.Add(face.MakeNullLinkService(face.MakeNullTransport()))
	// Create multicast faces based upon interfaces
	// TODO
	// Set up listeners
	udpListener, err := face.MakeUDPListener(face.MakeUDPFaceURI(4, "127.0.0.1", 6364))
	if err != nil {
		core.LogFatal("Main", err)
	}
	go udpListener.Run()

	// Set up signal handler channel and wait for interrupt
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, os.Kill)
	receivedSig := <-sigChannel
	core.LogInfo("Main", "Received signal", receivedSig, " - exiting")
	core.ShouldQuit = true

	// Wait for all forwarding threads to have quit
	for _, fw := range fw.Threads {
		<-fw.HasQuit
	}

	// Wait for all face threads to have quit
	/*for _, face := range face.Threads {
		<-face.HasQuit
	}*/
}
