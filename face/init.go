/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import "github.com/eric135/YaNFD/core"

// faceQueueSize is the maximum number of packets that can be buffered to be sent or received on a face.
var faceQueueSize int

// NDNEtherType is the standard EtherType for NDN.
var ndnEtherType int

// EthernetMulticastAddress is the standard multicast Ethernet URI for NDN.
var EthernetMulticastAddress string

// UDPUnicastPort is the standard unicast UDP port for NDN.
var UDPUnicastPort uint16

// UDPMulticastPort is the standard multicast UDP port for NDN.
var UDPMulticastPort uint16

// udp4MulticastAddress is the standard multicast UDP4 URI for NDN.
var udp4MulticastAddress string

// udp6MulticastAddress is the standard multicast UDP6 address for NDN.
var udp6MulticastAddress string

// UnixSocketPath is the standard Unix socket file path for NDN.
var UnixSocketPath string

// Configure configures the face system.
func Configure() {
	faceQueueSize = core.GetConfigIntDefault("faces.queue_size", 1024)
	ndnEtherType = core.GetConfigIntDefault("faces.ethernet.ethertype", 0x8624)
	EthernetMulticastAddress = core.GetConfigStringDefault("faces.ethernet.multicast_address", "01:00:5e:00:17:aa")
	UDPUnicastPort = core.GetConfigUint16Default("faces.udp.port_unicast", 6363)
	UDPMulticastPort = core.GetConfigUint16Default("faces.udp.port_multicast", 56363)
	udp4MulticastAddress = core.GetConfigStringDefault("faces.udp.multicast_address_ipv4", "224.0.23.170")
	udp6MulticastAddress = core.GetConfigStringDefault("faces.udp.multicast_address_ipv6", "ff02::114")
	UnixSocketPath = core.GetConfigStringDefault("faces.unix.socket_path", "/run/nfd.sock")
}
