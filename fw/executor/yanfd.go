/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package executor

import (
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/mgmt"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/table"
)

// YaNFDConfig is the configuration of YaNFD.
type YaNFDConfig struct {
	Version           string
	ConfigFileName    string
	DisableEthernet   bool
	DisableUnix       bool
	LogFile           string
	CpuProfile        string
	MemProfile        string
	BlockProfile      string
	MemoryBallastSize int
}

// YaNFD is the wrapper class for the NDN Forwarding Daemon.
// Note: only one instance of this class should be created.
type YaNFD struct {
	config *YaNFDConfig

	cpuProfileFile *os.File
	memProfileFile *os.File
	blockProfiler  *pprof.Profile

	unixListener *face.UnixStreamListener
	wsListener   *face.WebSocketListener
	tcpListeners []*face.TCPListener
}

// NewYaNFD creates a YaNFD. Don't call this function twice.
func NewYaNFD(config *YaNFDConfig) *YaNFD {
	// Provide metadata to other threads.
	core.Version = config.Version
	core.StartTimestamp = time.Now()

	// Allocate memory ballast (if enabled)
	if config.MemoryBallastSize > 0 {
		_ = make([]byte, config.MemoryBallastSize<<30)
	}

	// Initialize config file
	core.LoadConfig(config.ConfigFileName)
	core.InitializeLogger(config.LogFile)
	face.Configure()
	fw.Configure()
	table.Configure()
	mgmt.Configure()

	// Initialize profiling
	var cpuProfileFile *os.File
	var memProfileFile *os.File
	var blockProfiler *pprof.Profile
	var err error
	if config.CpuProfile != "" {
		cpuProfileFile, err = os.Create(config.CpuProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for CPU profile: ", err)
		}

		core.LogInfo("Main", "Profiling CPU - outputting to ", config.CpuProfile)
		pprof.StartCPUProfile(cpuProfileFile)
	}

	if config.BlockProfile != "" {
		core.LogInfo("Main", "Profiling blocking operations - outputting to ", config.BlockProfile)
		runtime.SetBlockProfileRate(1)
		blockProfiler = pprof.Lookup("block")
		// Output at end of runtime
	}

	return &YaNFD{
		config:         config,
		cpuProfileFile: cpuProfileFile,
		memProfileFile: memProfileFile,
		blockProfiler:  blockProfiler,
	}
}

// Start runs YaNFD. Note: this function may exit the program when there is error.
// This function is non-blocking.
func (y *YaNFD) Start() {
	core.LogInfo("Main", "Starting YaNFD")

	// Load strategies
	//core.LogInfo("Main", "Loading strategies")
	//fw.LoadStrategies()

	// Create null face
	nullFace := face.MakeNullLinkService(face.MakeNullTransport())
	face.FaceTable.Add(nullFace)
	go nullFace.Run(nil)

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
	ethEnabled := core.GetConfigBoolDefault("faces.ethernet.enabled", true) && !y.config.DisableEthernet
	tcpEnabled := core.GetConfigBoolDefault("faces.tcp.enabled", true)
	tcpPort := face.TCPUnicastPort
	y.tcpListeners = make([]*face.TCPListener, 0)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			core.LogInfo("Main", "Skipping interface ", iface.Name, " because not up")
			continue
		}

		if ethEnabled && iface.Flags&net.FlagMulticast != 0 {
			// Create multicast Ethernet face for interface
			multicastEthTransport, err := face.MakeMulticastEthernetTransport(multicastEthURI, ndn.MakeDevFaceURI(iface.Name))
			if err != nil {
				core.LogError("Main", "Unable to create MulticastEthernetTransport for ", iface.Name, ": ", err)
			} else {
				multicastEthFace := face.MakeNDNLPLinkService(multicastEthTransport, face.MakeNDNLPLinkServiceOptions())
				face.FaceTable.Add(multicastEthFace)
				faceCnt += 1
				go multicastEthFace.Run(nil)
				core.LogInfo("Main", "Created multicast Ethernet face for ", iface.Name)

				// Create Ethernet listener for interface
				// TODO
			}
		}

		// Create UDP/TCP listener and multicast UDP interface for every address on interface
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
				go multicastUDPFace.Run(nil)
				core.LogInfo("Main", "Created multicast UDP face for ", path, " on ", iface.Name)
			}

			udpListener, err := face.MakeUDPListener(ndn.MakeUDPFaceURI(ipVersion, path, face.UDPUnicastPort))
			if err != nil {
				core.LogError("Main", "Unable to create UDP listener for ", path, " on ", iface.Name, ": ", err)
				continue
			}
			faceCnt += 1
			go udpListener.Run()
			core.LogInfo("Main", "Created UDP listener for ", path, " on ", iface.Name)

			if tcpEnabled {
				tcpListener, err := face.MakeTCPListener(ndn.MakeTCPFaceURI(ipVersion, path, tcpPort))
				if err != nil {
					core.LogError("Main", "Unable to create TCP listener for ", path, " on ", iface.Name, ": ", err)
					continue
				}
				faceCnt += 1
				go tcpListener.Run()
				core.LogInfo("Main", "Created TCP listener for ", path, " on ", iface.Name)
				y.tcpListeners = append(y.tcpListeners, tcpListener)
			}
		}
	}

	if core.GetConfigBoolDefault("faces.unix.enabled", true) && !y.config.DisableUnix {
		// Set up Unix stream listener
		y.unixListener, err = face.MakeUnixStreamListener(ndn.MakeUnixFaceURI(face.UnixSocketPath))
		if err != nil {
			core.LogError("Main", "Unable to create Unix stream listener at ", face.UnixSocketPath, ": ", err)
		} else {
			faceCnt += 1
			go y.unixListener.Run()
			core.LogInfo("Main", "Created Unix stream listener for ", face.UnixSocketPath)
		}
	}

	if core.GetConfigBoolDefault("faces.websocket.enabled", true) {
		cfg := face.WebSocketListenerConfig{
			Bind:       core.GetConfigStringDefault("faces.websocket.bind", ""),
			Port:       core.GetConfigUint16Default("faces.websocket.port", 9696),
			TLSEnabled: core.GetConfigBoolDefault("faces.websocket.tls_enabled", false),
			TLSCert:    core.ResolveConfigFileRelPath(core.GetConfigStringDefault("faces.websocket.tls_cert", "")),
			TLSKey:     core.ResolveConfigFileRelPath(core.GetConfigStringDefault("faces.websocket.tls_key", "")),
		}
		y.wsListener, err = face.NewWebSocketListener(cfg)
		if err != nil {
			core.LogError("Main", "Unable to create ", cfg, ": ", err)
		} else {
			faceCnt++
			go y.wsListener.Run()
			core.LogInfo("Main", "Created ", cfg)
		}
	}

	if faceCnt <= 0 {
		core.LogFatal("Main", "No face or listener is successfully created. Quit.")
		os.Exit(2)
	}
}

// Stop shuts down YaNFD.
func (y *YaNFD) Stop() {
	core.LogInfo("Main", "Forwarder shutting down ...")
	core.ShouldQuit = true

	if y.config.MemProfile != "" {
		memProfileFile, err := os.Create(y.config.MemProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for memory profile: ", err)
		}

		core.LogInfo("Main", "Profiling memory - outputting to ", y.config.MemProfile)
		runtime.GC()
		if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
			core.LogFatal("Main", "Unable to write memory profile: ", err)
		}
	}

	// Wait for unix socket listener to quit
	if y.unixListener != nil {
		y.unixListener.Close()
		<-y.unixListener.HasQuit
	}
	if y.wsListener != nil {
		y.wsListener.Close()
	}

	// Wait for TCP listeners to quit
	for _, tcpListener := range y.tcpListeners {
		tcpListener.Close()
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

	// Shutdown Profilers
	if y.config.BlockProfile != "" {
		blockProfileFile, err := os.Create(y.config.BlockProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for block profile: ", err)
		}
		if err := y.blockProfiler.WriteTo(blockProfileFile, 0); err != nil {
			core.LogFatal("Main", "Unable to write block profile: ", err)
		}
		blockProfileFile.Close()
	}
	if y.config.MemProfile != "" {
		y.memProfileFile.Close()
	}
	if y.config.CpuProfile != "" {
		pprof.StopCPUProfile()
		y.cpuProfileFile.Close()
	}
}
