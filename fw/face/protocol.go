/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// NDNEtherType is the standard EtherType for NDN.
const NDNEtherType = 0x8624

// NDNMulticastEtherURI is the standard multicast Ethernet URI for NDN.
const NDNMulticastEtherURI = "ether://[01:00:5e:00:17:aa]"

// NDNUnicastUDPPort is the standard unicast UDP port for NDN.
const NDNUnicastUDPPort = 6363

// NDNMulticastUDPPort is the standard multicast UDP port for NDN.
const NDNMulticastUDPPort = 56363

// NDNMulticastUDP4URI is the standard multicast UDP4 URI for NDN.
const NDNMulticastUDP4URI = "udp4://224.0.23.170:56363"

// NDNMulticastUDP6Address is the standard multicast UDP6 address for NDN.
const NDNMulticastUDP6Address = "ff02::114"

// NDNUnixSocketFile is the standard Unix socket file for NDN.
const NDNUnixSocketFile = "/run/nfd.sock"
