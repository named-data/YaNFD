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
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"syscall"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/dispatch"
	"github.com/eric135/YaNFD/face"
	"github.com/eric135/YaNFD/fw"
	"github.com/eric135/YaNFD/mgmt"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/table"
)

// Version of YaNFD.
var Version string

// BuildTime contains the timestamp of when the version of YaNFD was built.
var BuildTime string

func main() {
	// Provide metadata to other threads.
	core.Version = Version
	core.BuildTime = BuildTime
	core.StartTimestamp = time.Now()

	// Parse command line options
	var shouldPrintVersion bool
	flag.BoolVar(&shouldPrintVersion, "version", false, "Print version and exit")
	var configFileName string
	flag.StringVar(&configFileName, "config", "/usr/local/etc/ndn/yanfd.toml", "Configuration file location")
	var disableEthernet bool
	flag.BoolVar(&disableEthernet, "disable-ethernet", false, "Disable Ethernet transports")
	var disableUnix bool
	flag.BoolVar(&disableUnix, "disable-unix", false, "Disable Unix stream transports")
	var cpuProfile string
	flag.StringVar(&cpuProfile, "cpu-profile", "", "Enable CPU profiling (output to specified file)")
	var memProfile string
	flag.StringVar(&memProfile, "mem-profile", "", "Enable memory profiling (output to specified file)")
	var memBalastSize uint
	flag.UintVar(&memBalastSize, "mem-balast", 4, "Size of memory balast in GB to avoid frequent GC (default 4GB)")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version " + core.Version + " (Built " + core.BuildTime + ")")
		fmt.Println("Copyright (C) 2020-2021 Eric Newberry")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	// Allocate memory "balast" to prevent frequent GC (https://blog.twitch.tv/en/2019/04/10/go-memory-ballast-how-i-learnt-to-stop-worrying-and-love-the-heap-26c2462549a2/)
	// This won't actually use up this many GB of RAM, since it'll all be in virtual memory as long as we don't reference it
	_ = make([]byte, memBalastSize<<30)

	// Initialize config file
	core.LoadConfig(configFileName)
	core.InitializeLogger()
	face.Configure()
	fw.Configure()
	table.Configure()

	if cpuProfile != "" {
		cpuProfileFile, err := os.Create(cpuProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for CPU profile: "+err.Error())
		}

		core.LogInfo("Main", "Profiling CPU - outputting to "+cpuProfile)
		pprof.StartCPUProfile(cpuProfileFile)
		defer cpuProfileFile.Close()
		defer pprof.StopCPUProfile()
	}

	if memProfile != "" {
		memProfileFile, err := os.Create(memProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for memory profile: "+err.Error())
		}

		core.LogInfo("Main", "Profiling memory - outputting to "+memProfile)
		runtime.GC()
		if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
			core.LogFatal("Main", "Unable to write memory profile: "+err.Error())
		}
		defer memProfileFile.Close()
	}

	core.LogInfo("Main", "Starting YaNFD")

	// Load strategies
	//core.LogInfo("Main", "Loading strategies")
	//fw.LoadStrategies()

	// Create null face
	nullFace := face.MakeNullLinkService(face.MakeNullTransport())
	face.FaceTable.Add(nullFace)
	go nullFace.Run()

	// Start management thread
	management := mgmt.MakeMgmtThread()
	go management.Run()

	// Create forwarding threads
	if fw.NumFwThreads < 1 || fw.NumFwThreads > fw.MaxFwThreads {
		core.LogFatal("Main", "Number of forwarding threads must be in range [1, "+strconv.Itoa(fw.MaxFwThreads)+"]")
	}
	fw.Threads = make(map[int]*fw.Thread)
	for i := 0; i < fw.NumFwThreads; i++ {
		newThread := fw.NewThread(i)
		fw.Threads[i] = newThread
		dispatch.AddFWThread(i, newThread)
		go fw.Threads[i].Run()
	}

	// Perform setup operations for each network interface
	ifaces, err := net.Interfaces()
	multicastEthURI := ndn.DecodeURIString("ether://[" + face.EthernetMulticastAddress + "]")
	if err != nil {
		core.LogFatal("Main", "Unable to access network interfaces: "+err.Error())
		os.Exit(2)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			core.LogInfo("Main", "Skipping interface "+iface.Name+" because not up")
			continue
		}

		if !disableEthernet && iface.Flags&net.FlagMulticast != 0 {
			// Create multicast Ethernet face for interface
			multicastEthTransport, err := face.MakeMulticastEthernetTransport(multicastEthURI, ndn.MakeDevFaceURI(iface.Name))
			if err != nil {
				core.LogFatal("Main", "Unable to create MulticastEthernetTransport for "+iface.Name+": "+err.Error())
				os.Exit(2)
			}
			multicastEthFace := face.MakeNDNLPLinkService(multicastEthTransport, face.MakeNDNLPLinkServiceOptions())
			face.FaceTable.Add(multicastEthFace)
			go multicastEthFace.Run()
			core.LogInfo("Main", "Created multicast Ethernet face for "+iface.Name)

			// Create Ethernet listener for interface
			// TODO
		}

		// Create UDP listener and multicast UDP interface for every address on interface
		addrs, err := iface.Addrs()
		if err != nil {
			core.LogFatal("Main", "Unable to access addresses on network interface "+iface.Name+": "+err.Error())
		}
		for _, addr := range addrs {
			ipAddr := addr.(*net.IPNet)

			ipVersion := 4
			path := ipAddr.IP.String()
			if ipAddr.IP.To4() == nil {
				ipVersion = 6
				path += "%" + iface.Name
			}

			if !addr.(*net.IPNet).IP.IsLoopback() {
				multicastUDPTransport, err := face.MakeMulticastUDPTransport(ndn.MakeUDPFaceURI(ipVersion, path, face.UDPMulticastPort))
				if err != nil {
					core.LogFatal("Main", "Unable to create MulticastUDPTransport for "+path+" on "+iface.Name+": "+err.Error())
					os.Exit(2)
				}
				multicastUDPFace := face.MakeNDNLPLinkService(multicastUDPTransport, face.MakeNDNLPLinkServiceOptions())
				face.FaceTable.Add(multicastUDPFace)
				go multicastUDPFace.Run()
				core.LogInfo("Main", "Created multicast UDP face for "+path+" on "+iface.Name)
			}

			udpListener, err := face.MakeUDPListener(ndn.MakeUDPFaceURI(ipVersion, path, 6363))
			if err != nil {
				core.LogFatal("Main", "Unable to create UDP listener for "+path+" on "+iface.Name+": "+err.Error())
				os.Exit(2)
			}
			go udpListener.Run()
			core.LogInfo("Main", "Created UDP listener for "+path+" on "+iface.Name)
		}
	}

	var unixListener *face.UnixStreamListener
	if !disableUnix {
		// Set up Unix stream listener
		unixListener, err = face.MakeUnixStreamListener(ndn.MakeUnixFaceURI(face.UnixSocketPath))
		if err != nil {
			core.LogFatal("Main", "Unable to create Unix stream listener at "+face.UnixSocketPath+": "+err.Error())
			os.Exit(2)
		}
		go unixListener.Run()
		core.LogInfo("Main", "Created Unix stream listener for "+face.UnixSocketPath)
	}

	// Set up signal handler channel and wait for interrupt
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	core.LogInfo("Main", "Received signal "+receivedSig.String()+" - exiting")
	core.ShouldQuit = true

	// Wait for unix socket listener to quit
	if !disableUnix {
		unixListener.Close()
		<-unixListener.HasQuit
	}

	// Tell all faces to quit
	for _, face := range face.FaceTable.Faces {
		face.Close()
	}

	// Wait for all faces to quit
	for _, face := range face.FaceTable.Faces {
		//core.LogTrace("Main", "Waiting for face "+strconv.Itoa(face.FaceID())+" to quit")
		core.LogTrace("Main", "Waiting for face "+face.String()+" to quit")
		<-face.GetHasQuit()
	}

	// Tell all forwarding threads to quit
	for _, fw := range fw.Threads {
		fw.TellToQuit()
	}

	// Wait for all forwarding threads to have quit
	for _, fw := range fw.Threads {
		<-fw.HasQuit
	}
}
