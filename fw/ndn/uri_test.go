/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_test

import (
	"net"
	"testing"

	"github.com/named-data/YaNFD/ndn"
	"github.com/stretchr/testify/assert"
)

func TestDev(t *testing.T) {
	uri := ndn.MakeDevFaceURI("lo")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
	assert.Equal(t, "lo", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "dev://lo", uri.String())

	uri = ndn.MakeDevFaceURI("fakeif")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())

	uri = ndn.DecodeURIString("dev://lo")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
	assert.Equal(t, "lo", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "dev://lo", uri.String())

	uri = ndn.DecodeURIString("dev://fakeif")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
}

func TestEthernet(t *testing.T) {
	mac, err := net.ParseMAC("00:11:22:33:44:AA")
	assert.NoError(t, err)
	uri := ndn.MakeEthernetFaceURI(mac)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "ether", uri.Scheme())
	assert.Equal(t, "00:11:22:33:44:aa", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "ether://[00:11:22:33:44:aa]", uri.String())

	uri = ndn.DecodeURIString("ether://[00:11:22:33:44:AA]")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "ether", uri.Scheme())
	assert.Equal(t, "00:11:22:33:44:aa", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "ether://[00:11:22:33:44:aa]", uri.String())
}

func TestFD(t *testing.T) {
	uri := ndn.MakeFDFaceURI(27)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "fd", uri.Scheme())
	assert.Equal(t, "27", uri.Path())
	assert.Equal(t, "27", uri.PathHost())
	assert.Equal(t, uint16(0), uri.Port())

	uri = ndn.DecodeURIString("fd://27")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "fd", uri.Scheme())
	assert.Equal(t, "27", uri.Path())
	assert.Equal(t, "27", uri.PathHost())
	assert.Equal(t, uint16(0), uri.Port())

	uri = ndn.DecodeURIString("fd://27a")
	assert.False(t, uri.IsCanonical())

	uri = ndn.DecodeURIString("fd://27:6363")
	assert.False(t, uri.IsCanonical())
}

func TestNull(t *testing.T) {
	uri := ndn.MakeNullFaceURI()
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "null", uri.Scheme())
	assert.Equal(t, "", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "null://", uri.String())

	uri = ndn.DecodeURIString("null://")
	assert.Equal(t, "null", uri.Scheme())
	assert.Equal(t, "", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "null://", uri.String())
}

func TestUDP(t *testing.T) {
	uri := ndn.MakeUDPFaceURI(4, "192.0.2.1", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "192.0.2.1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://192.0.2.1:6363", uri.String())

	uri = ndn.MakeUDPFaceURI(4, "[2001:db8::1]", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())      // Canonized into UDP6
	assert.Equal(t, "2001:db8::1", uri.Path()) // Braces are trimmed by canonization
	assert.Equal(t, uint16(6363), uri.Port())
	assert.NoError(t, uri.Canonize())
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1]:6363", uri.String())

	uri = ndn.DecodeURIString("udp4://192.0.2.1:6363")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "192.0.2.1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://192.0.2.1:6363", uri.String())

	uri = ndn.MakeUDPFaceURI(6, "2001:db8::1", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1]:6363", uri.String())

	uri = ndn.MakeUDPFaceURI(6, "2001:db8::1%eth0", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1%eth0", uri.Path())
	assert.Equal(t, "2001:db8::1", uri.PathHost())
	assert.Equal(t, "eth0", uri.PathZone())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1%eth0]:6363", uri.String())

	uri = ndn.MakeUDPFaceURI(6, "192.0.2.1", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "192.0.2.1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.NoError(t, uri.Canonize())
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "192.0.2.1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://192.0.2.1:6363", uri.String())

	uri = ndn.DecodeURIString("udp6://[2001:db8::1]:6363")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1]:6363", uri.String())

	// Test DNS resolution
	uri = ndn.MakeUDPFaceURI(4, "named-data.net", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4://104.154.51.235:6363", uri.String())
	assert.NoError(t, uri.Canonize())
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "104.154.51.235", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://104.154.51.235:6363", uri.String())
}

func TestUnix(t *testing.T) {
	uri := ndn.MakeUnixFaceURI("/run/nfd.sock")
	assert.True(t, uri.IsCanonical())
	assert.NoError(t, uri.Canonize())
	assert.True(t, uri.IsCanonical())

	// Is a directory
	uri = ndn.MakeUnixFaceURI("/run")
	assert.False(t, uri.IsCanonical())
	assert.Error(t, uri.Canonize())
	assert.False(t, uri.IsCanonical())
}

func TestUnknown(t *testing.T) {
	uri := ndn.DecodeURIString("fake://abc:123")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "unknown://", uri.String())
	assert.Error(t, uri.Canonize())
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "unknown://", uri.String())
}
