/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/eric135/YaNFD/face"
	"github.com/stretchr/testify/assert"
)

func TestDev(t *testing.T) {
	uri := face.MakeDevFaceURI("lo")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
	assert.Equal(t, "lo", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "dev://lo", uri.String())

	uri = face.MakeDevFaceURI("fakeif")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())

	uri = face.DecodeURIString("dev://lo")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
	assert.Equal(t, "lo", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "dev://lo", uri.String())

	uri = face.DecodeURIString("dev://fakeif")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "dev", uri.Scheme())
}

func TestEthernet(t *testing.T) {
	mac, err := net.ParseMAC("00:11:22:33:44:AA")
	assert.NoError(t, err)
	uri := face.MakeEthernetFaceURI(mac)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "eth", uri.Scheme())
	assert.Equal(t, "00:11:22:33:44:aa", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "eth://[00:11:22:33:44:aa]", uri.String())

	uri = face.DecodeURIString("eth://[00:11:22:33:44:AA]")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "eth", uri.Scheme())
	assert.Equal(t, "00:11:22:33:44:aa", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "eth://[00:11:22:33:44:aa]", uri.String())
}

func testFD(t *testing.T) {
	uri := face.MakeFDFaceURI(27)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "fd", uri.Scheme())
	assert.Equal(t, "27", uri.Path())
	assert.Equal(t, "27", uri.PathHost())
	assert.Equal(t, 0, uri.Port())

	uri = face.DecodeURIString("fd://27")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "fd", uri.Scheme())
	assert.Equal(t, "27", uri.Path())
	assert.Equal(t, "27", uri.PathHost())
	assert.Equal(t, 0, uri.Port())

	uri = face.DecodeURIString("fd://27a")
	assert.False(t, uri.IsCanonical())

	uri = face.DecodeURIString("fd://27:6363")
	assert.False(t, uri.IsCanonical())
}

func TestNull(t *testing.T) {
	uri := face.MakeNullFaceURI()
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "null", uri.Scheme())
	assert.Equal(t, "", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "null://", uri.String())

	uri = face.DecodeURIString("null://")
	assert.Equal(t, "null", uri.Scheme())
	assert.Equal(t, "", uri.Path())
	assert.Equal(t, uint16(0), uri.Port())
	assert.Equal(t, "null://", uri.String())
}

func TestUDP(t *testing.T) {
	uri := face.MakeUDPFaceURI(4, "192.0.2.1", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "192.0.2.1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://192.0.2.1:6363", uri.String())

	uri = face.MakeUDPFaceURI(4, "[2001:db8::1]", 6363)
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

	uri = face.DecodeURIString("udp4://192.0.2.1:6363")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp4", uri.Scheme())
	assert.Equal(t, "192.0.2.1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp4://192.0.2.1:6363", uri.String())

	uri = face.MakeUDPFaceURI(6, "2001:db8::1", 6363)
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1]:6363", uri.String())

	fmt.Println("A")
	uri = face.MakeUDPFaceURI(6, "2001:db8::1%eth0", 6363)
	fmt.Println("B")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1%eth0", uri.Path())
	assert.Equal(t, "2001:db8::1", uri.PathHost())
	assert.Equal(t, "eth0", uri.PathZone())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1%eth0]:6363", uri.String())

	uri = face.MakeUDPFaceURI(6, "192.0.2.1", 6363)
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

	uri = face.DecodeURIString("udp6://[2001:db8::1]:6363")
	assert.True(t, uri.IsCanonical())
	assert.Equal(t, "udp6", uri.Scheme())
	assert.Equal(t, "2001:db8::1", uri.Path())
	assert.Equal(t, uint16(6363), uri.Port())
	assert.Equal(t, "udp6://[2001:db8::1]:6363", uri.String())

	// Test DNS resolution
	uri = face.MakeUDPFaceURI(4, "named-data.net", 6363)
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
	uri := face.MakeUnixFaceURI("/run/nfd.sock")
	assert.True(t, uri.IsCanonical())
	assert.NoError(t, uri.Canonize())
	assert.True(t, uri.IsCanonical())

	// Is a directory
	uri = face.MakeUnixFaceURI("/run")
	assert.False(t, uri.IsCanonical())
	assert.Error(t, uri.Canonize())
	assert.False(t, uri.IsCanonical())
}

func TestUnknown(t *testing.T) {
	uri := face.DecodeURIString("fake://abc:123")
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "unknown://", uri.String())
	assert.Error(t, uri.Canonize())
	assert.False(t, uri.IsCanonical())
	assert.Equal(t, "unknown://", uri.String())
}
