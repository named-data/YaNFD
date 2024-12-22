/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"os"
	"time"

	"github.com/pulsejet/ndnd/fw/core"
)

// faceQueueSize is the maximum number of packets that can be buffered to be sent or received on a face.
var faceQueueSize int

// congestionMarking indicates whether congestion marking is enabled or disabled.
var congestionMarking bool

// lockThreadsToCores determines whether face threads will be locked to logical cores.
var lockThreadsToCores bool

// UDPUnicastPort is the standard unicast UDP port for NDN.
var UDPUnicastPort uint16

// UDPMulticastPort is the standard multicast UDP port for NDN.
var UDPMulticastPort uint16

// udp4MulticastAddress is the standard multicast UDP4 URI for NDN.
var udp4MulticastAddress string

// udp6MulticastAddress is the standard multicast UDP6 address for NDN.
var udp6MulticastAddress string

// udpLifetime is the lifetime of on-demand UDP faces after they become idle.
var udpLifetime time.Duration

// TCPUnicastPort is the standard unicast TCP port for NDN.
var TCPUnicastPort uint16

// tcpLifetime is the lifetime of on-demand TCP faces after they become idle.
var tcpLifetime time.Duration

// UnixSocketPath is the standard Unix socket file path for NDN.
var UnixSocketPath string

// Configure configures the face system.
func Configure() {
	faceQueueSize = core.GetConfig().Faces.QueueSize
	congestionMarking = core.GetConfig().Faces.CongestionMarking
	lockThreadsToCores = core.GetConfig().Faces.LockThreadsToCores
	UDPUnicastPort = core.GetConfig().Faces.Udp.PortUnicast
	TCPUnicastPort = core.GetConfig().Faces.Tcp.PortUnicast
	UDPMulticastPort = core.GetConfig().Faces.Udp.PortMulticast
	udp4MulticastAddress = core.GetConfig().Faces.Udp.MulticastAddressIpv4
	udp6MulticastAddress = core.GetConfig().Faces.Udp.MulticastAddressIpv6
	udpLifetime = time.Duration(core.GetConfig().Faces.Udp.Lifetime) * time.Second
	tcpLifetime = time.Duration(core.GetConfig().Faces.Tcp.Lifetime) * time.Second
	UnixSocketPath = os.ExpandEnv(core.GetConfig().Faces.Unix.SocketPath)
}
