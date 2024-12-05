/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_test

import (
	"net"
	"net/url"
	"testing"

	"github.com/named-data/YaNFD/ndn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDev(t *testing.T) {
	uri := ndn.MakeDevFaceURI("lo")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
	assert.Equal(t, "lo", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "dev://lo", uri.String())

	uri = ndn.MakeDevFaceURI("fakeif")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())

	uri = ndn.DecodeURIString("dev://lo")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
	assert.Equal(t, "lo", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "dev://lo", uri.String())

	uri = ndn.DecodeURIString("dev://fakeif")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
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
	// assert.NoError(t, uri.Canonize())
	// assert.True(t, uri.IsCanonical())
	// assert.Equal(t, "udp6", uri.Scheme())
	// assert.Equal(t, "2001:db8::1", uri.Path())
	// assert.Equal(t, uint16(6363), uri.Port())
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
	// assert.NoError(t, uri.Canonize())
	// assert.True(t, uri.IsCanonical())
	// assert.Equal(t, "udp4", uri.Scheme())
	// assert.Equal(t, "192.0.2.1", uri.Path())
	// assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://192.0.2.1:6363", uri.String())

	uri = ndn.DecodeURIString("udp6://[2001:db8::1]:6363")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1]:6363", uri.String())

	// Test DNS resolution
	// uri = ndn.MakeUDPFaceURI(4, "named-data.net", 6363)
	// assert.True(t, uri.IsCanonical())
	// assert.Equal(t, "udp4://104.154.51.235:6363", uri.String())
	// assert.NoError(t, uri.Canonize())
	// assert.True(t, uri.IsCanonical())
	// assert.Equal(t, "udp4", uri.Scheme())
	// assert.Equal(t, "104.154.51.235", uri.Path())
	// assert.Equal(t, uint16(6363), uri.Port())
	// assert.Equal(t, "udp4://104.154.51.235:6363", uri.String())
}

func TestUnix(t *testing.T) {
	uri := ndn.MakeUnixFaceURI("/run/nfd/nfd.sock")
	assert.True(t, uri.IsCanonical())
	// assert.NoError(t, uri.Canonize())
	// assert.True(t, uri.IsCanonical())

	// Is a directory
	uri = ndn.MakeUnixFaceURI("/run")
	assert.True(t, uri.IsCanonical())
	// assert.Error(t, uri.Canonize())
	// assert.False(t, uri.IsCanonical())
}

func TestWebSocket(t *testing.T) {
	{
		u, e := url.Parse("wss://:8443")
		require.NoError(t, e)
		uri := ndn.MakeWebSocketServerFaceURI(u)
		require.NotNil(t, uri)
		assert.Equal(t, "wss", uri.Scheme())
		assert.Equal(t, "", uri.Path())
		assert.Equal(t, uint16(8443), uri.Port())
		assert.Equal(t, "wss://:8443", uri.String())

		uri = ndn.DecodeURIString("wss://:8443")
		assert.NotNil(t, uri)
	}

	{
		u, e := url.Parse("ws://[::1]:9696")
		require.NoError(t, e)
		uri := ndn.MakeWebSocketServerFaceURI(u)
		require.NotNil(t, uri)
		assert.Equal(t, "ws", uri.Scheme())
		assert.Equal(t, "::1", uri.Path())
		assert.Equal(t, uint16(9696), uri.Port())
		assert.Equal(t, "ws://[::1]:9696", uri.String())

		uri = ndn.DecodeURIString("ws://[::1]:9696")
		assert.NotNil(t, uri)
	}

	{
		addr := &net.TCPAddr{
			IP:   net.ParseIP("2001:db8:3334:7d::566b"),
			Port: 59505,
		}
		uri := ndn.MakeWebSocketClientFaceURI(addr)
		require.NotNil(t, uri)
		assert.Equal(t, "wsclient", uri.Scheme())
		assert.Equal(t, "2001:db8:3334:7d::566b", uri.Path())
		assert.Equal(t, uint16(59505), uri.Port())
		assert.Equal(t, "wsclient://[2001:db8:3334:7d::566b]:59505", uri.String())

		uri = ndn.DecodeURIString("wsclient://[2001:db8:3334:7d::566b]:59505")
		assert.NotNil(t, uri)
	}
}

func TestUnknown(t *testing.T) {
	uri := ndn.DecodeURIString("fake://abc:123")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "unknown://", uri.String())
	assert.Error(t, uri.Canonize())
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "unknown://", uri.String())
}
