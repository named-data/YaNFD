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
	"time"

	"github.com/named-data/ndnd/fw/core"
	"github.com/named-data/ndnd/fw/defn"
	"github.com/named-data/ndnd/fw/dispatch"
	"github.com/named-data/ndnd/fw/face"
	"github.com/named-data/ndnd/fw/fw"
	"github.com/named-data/ndnd/fw/mgmt"
	"github.com/named-data/ndnd/fw/table"
)

// YaNFDConfig is the configuration of YaNFD.
type YaNFDConfig struct {
	Version string
	LogFile string

	Config  *core.Config
	BaseDir string

	CpuProfile        string
	MemProfile        string
	BlockProfile      string
	MemoryBallastSize int
}

// YaNFD is the wrapper class for the NDN Forwarding Daemon.
// Note: only one instance of this class should be created.
type YaNFD struct {
	config   *YaNFDConfig
	profiler *Profiler

	unixListener *face.UnixStreamListener
	wsListener   *face.WebSocketListener
	tcpListeners []*face.TCPListener
	udpListener  *face.UDPListener
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
	core.LoadConfig(config.Config, config.BaseDir)
	core.InitializeLogger(config.LogFile)
	face.Configure()
	fw.Configure()
	table.Configure()
	mgmt.Configure()

	return &YaNFD{
		config:   config,
		profiler: NewProfiler(config),
	}
}

// Start runs YaNFD. Note: this function may exit the program when there is error.
// This function is non-blocking.
func (y *YaNFD) Start() {
	core.LogInfo("Main", "Starting YaNFD")

	// Start profiler
	y.profiler.Start()

	// Initialize FIB table
	fibTableAlgorithm := core.GetConfig().Tables.Fib.Algorithm
	table.CreateFIBTable(fibTableAlgorithm)

	// Create null face
	face.MakeNullLinkService(face.MakeNullTransport()).Run(nil)

	// Start management thread
	go mgmt.MakeMgmtThread().Run()

	// Create forwarding threads
	if fw.NumFwThreads < 1 || fw.NumFwThreads > fw.MaxFwThreads {
		core.LogFatal("Main", "Number of forwarding threads must be in range [1, ", fw.MaxFwThreads, "]")
		os.Exit(2)
	}

	fw.Threads = make([]*fw.Thread, fw.NumFwThreads)
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
	if err != nil {
		core.LogFatal("Main", "Unable to access network interfaces: ", err)
		os.Exit(2)
	}
	tcpEnabled := core.GetConfig().Faces.Tcp.Enabled
	tcpPort := face.TCPUnicastPort
	y.tcpListeners = make([]*face.TCPListener, 0)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			core.LogInfo("Main", "Skipping interface ", iface.Name, " because not up")
			continue
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
				multicastUDPTransport, err := face.MakeMulticastUDPTransport(
					defn.MakeUDPFaceURI(ipVersion, path, face.UDPMulticastPort))
				if err != nil {
					core.LogError("Main", "Unable to create MulticastUDPTransport for ", path, " on ", iface.Name, ": ", err)
					continue
				}

				face.MakeNDNLPLinkService(
					multicastUDPTransport,
					face.MakeNDNLPLinkServiceOptions(),
				).Run(nil)

				faceCnt += 1
				core.LogInfo("Main", "Created multicast UDP face for ", path, " on ", iface.Name)
			}

			udpListener, err := face.MakeUDPListener(defn.MakeUDPFaceURI(ipVersion, path, face.UDPUnicastPort))
			if err != nil {
				core.LogError("Main", "Unable to create UDP listener for ", path, " on ", iface.Name, ": ", err)
				continue
			}
			faceCnt += 1
			go udpListener.Run()
			y.udpListener = udpListener
			core.LogInfo("Main", "Created UDP listener for ", path, " on ", iface.Name)

			if tcpEnabled {
				tcpListener, err := face.MakeTCPListener(defn.MakeTCPFaceURI(ipVersion, path, tcpPort))
				if err != nil {
					core.LogError("Main", "Unable to create TCP listener for ", path, " on ", iface.Name, ": ", err)
					continue
				}
				faceCnt += 1
				go tcpListener.Run()
				y.tcpListeners = append(y.tcpListeners, tcpListener)
				core.LogInfo("Main", "Created TCP listener for ", path, " on ", iface.Name)
			}
		}
	}
	if core.GetConfig().Faces.Unix.Enabled {
		// Set up Unix stream listener
		y.unixListener, err = face.MakeUnixStreamListener(defn.MakeUnixFaceURI(face.UnixSocketPath))
		if err != nil {
			core.LogError("Main", "Unable to create Unix stream listener at ", face.UnixSocketPath, ": ", err)
		} else {
			faceCnt += 1
			go y.unixListener.Run()
			core.LogInfo("Main", "Created Unix stream listener for ", face.UnixSocketPath)
		}
	}

	if core.GetConfig().Faces.WebSocket.Enabled {
		cfg := face.WebSocketListenerConfig{
			Bind:       core.GetConfig().Faces.WebSocket.Bind,
			Port:       core.GetConfig().Faces.WebSocket.Port,
			TLSEnabled: core.GetConfig().Faces.WebSocket.TlsEnabled,
			TLSCert:    core.ResolveConfigFileRelPath(core.GetConfig().Faces.WebSocket.TlsCert),
			TLSKey:     core.ResolveConfigFileRelPath(core.GetConfig().Faces.WebSocket.TlsKey),
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

	// Stop profiler
	y.profiler.Stop()

	// Wait for unix socket listener to quit
	if y.unixListener != nil {
		y.unixListener.Close()
	}
	if y.wsListener != nil {
		y.wsListener.Close()
	}

	// Wait for UDP listener to quit
	if y.udpListener != nil {
		y.udpListener.Close()
	}

	// Wait for TCP listeners to quit
	for _, tcpListener := range y.tcpListeners {
		tcpListener.Close()
	}

	// Tell all faces to quit
	for _, face := range face.FaceTable.GetAll() {
		face.Close()
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
