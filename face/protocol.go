/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// NDNEtherType is the standard EtherType for NDN.
const NDNEtherType = 0x8624

// NDNUnicastUDPPort is the standard unicast UDP port for NDN.
const NDNUnicastUDPPort = 6363

// NDNMulticastUDPPort is the standard multicast UDP port for NDN.
const NDNMulticastUDPPort = 56363

// NDNMulticastUDP4URI is the standard multicast UDP4 URI for NDN.
const NDNMulticastUDP4URI = "udp4://224.0.23.170:56363"

// NDNMulticastUDP6URI is the standard multicast UDP6 URI for NDN.
const NDNMulticastUDP6URI = "udp6://[ff02::114]:56363"

// NDNUnixSocketFile is the standard Unix socket file for NDN.
const NDNUnixSocketFile = "/run/nfd.sock"
