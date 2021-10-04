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
	"syscall"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/mgmt"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"
)

// Version of YaNFD.
var Version string

func main() {
	// Provide metadata to other threads.
	core.Version = Version
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
	var blockProfile string
	flag.StringVar(&blockProfile, "block-profile", "", "Enable block profiling (output to specified file")
	var memoryBallastSize int
	flag.IntVar(&memoryBallastSize, "memory-ballast", 0, "Enable memory ballast of specified size (in GB) to avoid frequent garbage collection")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version " + core.Version)
		fmt.Println("Copyright (C) 2020-2021 Eric Newberry")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	// Allocate memory ballast (if enabled)
	if memoryBallastSize > 0 {
		_ = make([]byte, memoryBallastSize<<30)
	}

	// Initialize config file
	core.LoadConfig(configFileName)
	core.InitializeLogger()
	face.Configure()
	fw.Configure()
	table.Configure()
	mgmt.Configure()

	if cpuProfile != "" {
		cpuProfileFile, err := os.Create(cpuProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for CPU profile: ", err)
		}

		core.LogInfo("Main", "Profiling CPU - outputting to ", cpuProfile)
		pprof.StartCPUProfile(cpuProfileFile)
		defer cpuProfileFile.Close()
		defer pprof.StopCPUProfile()
	}

	if memProfile != "" {
		memProfileFile, err := os.Create(memProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for memory profile: ", err)
		}

		core.LogInfo("Main", "Profiling memory - outputting to ", memProfile)
		runtime.GC()
		if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
			core.LogFatal("Main", "Unable to write memory profile: ", err)
		}
		defer memProfileFile.Close()
	}

	if blockProfile != "" {
		core.LogInfo("Main", "Profiling blocking operations - outputting to ", blockProfile)
		runtime.SetBlockProfileRate(1)
		blockProfiler := pprof.Lookup("block")
		// Output at end of runtime
		defer func() {
			blockProfileFile, err := os.Create(blockProfile)
			if err != nil {
				core.LogFatal("Main", "Unable to open output file for block profile: ", err)
			}
			if err := blockProfiler.WriteTo(blockProfileFile, 0); err != nil {
				core.LogFatal("Main", "Unable to write block profile: ", err)
			}
			blockProfileFile.Close()
		}()
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
		core.LogFatal("Main", "Number of forwarding threads must be in range [1, ", fw.MaxFwThreads, "]")
	}
	fw.Threads = make(map[int]*fw.Thread)
	var fwForDispatch []dispatch.FWThread
	for i := 0; i < fw.NumFwThreads; i++ {
		newThread := fw.NewThread(i)
		fw.Threads[i] = newThread
		fwForDispatch = append(fwForDispatch, newThread)
		go fw.Threads[i].Run()
	}
	dispatch.InitializeFWThreads(fwForDispatch)

	// Perform setup operations for each network interface
	faceCnt := 0
	ifaces, err := net.Interfaces()
	multicastEthURI := ndn.DecodeURIString("ether://[" + face.EthernetMulticastAddress + "]")
	if err != nil {
		core.LogFatal("Main", "Unable to access network interfaces: ", err)
		os.Exit(2)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			core.LogInfo("Main", "Skipping interface ", iface.Name, " because not up")
			continue
		}

		if !disableEthernet && iface.Flags&net.FlagMulticast != 0 {
			// Create multicast Ethernet face for interface
			multicastEthTransport, err := face.MakeMulticastEthernetTransport(multicastEthURI, ndn.MakeDevFaceURI(iface.Name))
			if err != nil {
				core.LogError("Main", "Unable to create MulticastEthernetTransport for ", iface.Name, ": ", err)
			} else {
				multicastEthFace := face.MakeNDNLPLinkService(multicastEthTransport, face.MakeNDNLPLinkServiceOptions())
				face.FaceTable.Add(multicastEthFace)
				faceCnt += 1
				go multicastEthFace.Run()
				core.LogInfo("Main", "Created multicast Ethernet face for ", iface.Name)

				// Create Ethernet listener for interface
				// TODO
			}
		}

		// Create UDP listener and multicast UDP interface for every address on interface
		addrs, err := iface.Addrs()
		if err != nil {
			core.LogFatal("Main", "Unable to access addresses on network interface ", iface.Name, ": ", err)
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
					core.LogError("Main", "Unable to create MulticastUDPTransport for ", path, " on ", iface.Name, ": ", err)
					continue
				}
				multicastUDPFace := face.MakeNDNLPLinkService(multicastUDPTransport, face.MakeNDNLPLinkServiceOptions())
				face.FaceTable.Add(multicastUDPFace)
				faceCnt += 1
				go multicastUDPFace.Run()
				core.LogInfo("Main", "Created multicast UDP face for ", path, " on ", iface.Name)
			}

			udpListener, err := face.MakeUDPListener(ndn.MakeUDPFaceURI(ipVersion, path, 6363))
			if err != nil {
				core.LogError("Main", "Unable to create UDP listener for ", path, " on ", iface.Name, ": ", err)
				continue
			}
			faceCnt += 1
			go udpListener.Run()
			core.LogInfo("Main", "Created UDP listener for ", path, " on ", iface.Name)
		}
	}

	var unixListener *face.UnixStreamListener
	if !disableUnix {
		// Set up Unix stream listener
		unixListener, err = face.MakeUnixStreamListener(ndn.MakeUnixFaceURI(face.UnixSocketPath))
		if err != nil {
			core.LogError("Main", "Unable to create Unix stream listener at ", face.UnixSocketPath, ": ", err)
		} else {
			faceCnt += 1
			go unixListener.Run()
			core.LogInfo("Main", "Created Unix stream listener for ", face.UnixSocketPath)
		}
	}

	var wsListener *face.WebSocketListener
	if core.GetConfigBoolDefault("faces.websocket.enabled", true) {
		cfg := face.WebSocketListenerConfig{
			Bind:       core.GetConfigStringDefault("faces.websocket.bind", ""),
			Port:       core.GetConfigUint16Default("faces.websocket.port", 9696),
			TLSEnabled: core.GetConfigBoolDefault("faces.websocket.tls_enabled", false),
			TLSCert:    core.ResolveConfigFileRelPath(core.GetConfigStringDefault("faces.websocket.tls_cert", "")),
			TLSKey:     core.ResolveConfigFileRelPath(core.GetConfigStringDefault("faces.websocket.tls_key", "")),
		}
		wsListener, err = face.NewWebSocketListener(cfg)
		if err != nil {
			core.LogError("Main", "Unable to create ", cfg, ": ", err)
		} else {
			faceCnt++
			go wsListener.Run()
			core.LogInfo("Main", "Created ", cfg)
		}
	}

	if faceCnt <= 0 {
		core.LogFatal("Main", "No face or listener is successfully created. Quit.")
		os.Exit(2)
	}

	// Set up signal handler channel and wait for interrupt
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	receivedSig := <-sigChannel
	core.LogInfo("Main", "Received signal ", receivedSig, " - exiting")
	core.ShouldQuit = true

	// Wait for unix socket listener to quit
	if !disableUnix {
		unixListener.Close()
		<-unixListener.HasQuit
	}
	if wsListener != nil {
		wsListener.Close()
	}

	// Tell all faces to quit
	for _, face := range face.FaceTable.Faces {
		face.Close()
	}

	// Wait for all faces to quit
	for _, face := range face.FaceTable.Faces {
		core.LogTrace("Main", "Waiting for face ", face, " to quit")
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
