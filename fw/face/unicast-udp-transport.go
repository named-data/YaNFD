// +build linux darwin
/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"net"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/util"
	"golang.org/x/sys/unix"
)

// UnicastUDPTransport is a unicast UDP transport.
type UnicastUDPTransport struct {
	socket     int
	remoteAddr unix.Sockaddr
	transportBase
}

// MakeUnicastUDPTransport creates a new unicast UDP transport.
func MakeUnicastUDPTransport(remoteURI URI, localURI URI) (*UnicastUDPTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || (remoteURI.Scheme() != "udp4" && remoteURI.Scheme() != "udp6") || !localURI.IsCanonical() || remoteURI.Scheme() != localURI.Scheme() {
		return nil, core.ErrNotCanonical
	}

	var t UnicastUDPTransport
	t.makeTransportBase(remoteURI, localURI, core.MaxNDNPacketSize)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = Local
	} else {
		t.scope = NonLocal
	}

	// Attempt to connect to remote URI
	var err error
	if t.localURI.Scheme() == "udp4" {
		t.remoteAddr = new(unix.SockaddrInet4)
		copy(t.remoteAddr.(*unix.SockaddrInet4).Addr[0:3], net.ParseIP(remoteURI.Path()))
		t.remoteAddr.(*unix.SockaddrInet4).Port = int(remoteURI.Port())
		t.socket, err = unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	} else if t.localURI.Scheme() == "udp6" {
		t.remoteAddr = new(unix.SockaddrInet6)
		copy(t.remoteAddr.(*unix.SockaddrInet6).Addr[0:15], net.ParseIP(remoteURI.Path()))
		t.remoteAddr.(*unix.SockaddrInet6).Port = int(remoteURI.Port())
		t.socket, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	}
	if err != nil {
		return nil, err
	}

	err = unix.SetsockoptInt(t.socket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		core.LogError(t, "Unable to allow address reuse:", err)
		unix.Close(t.socket)
		return nil, err
	}

	if t.localURI.Scheme() == "udp4" {
		var localAddr unix.SockaddrInet4
		copy(localAddr.Addr[0:3], net.ParseIP(localURI.Path()))
		localAddr.Port = int(localURI.Port())
		err = unix.Bind(t.socket, &localAddr)
	} else if t.localURI.Scheme() == "udp6" {
		var localAddr unix.SockaddrInet6
		copy(localAddr.Addr[0:15], net.ParseIP(localURI.Path()))
		localAddr.Port = int(localURI.Port())
		err = unix.Bind(t.socket, &localAddr)
	}

	if err != nil {
		unix.Close(t.socket)
		return nil, err
	}

	return &t, nil
}

func (t *UnicastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	err := unix.Sendto(t.socket, frame, 0, t.remoteAddr)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP and Face DOWN")
		t.changeState(Down)
		t.hasQuit <- true
	}
}

func (t *UnicastUDPTransport) runReceive() {
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit && t.state != Down {
		readSize, _, err := unix.Recvfrom(t.socket, recvBuf, 0)
		if err != nil {
			core.LogWarn(t, "Unable to read from socket (", err, ") - DROP and Face DOWN")
			t.changeState(Down)
			break
		}

		core.LogTrace(t, "Receive of size", readSize)

		if readSize > core.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
		}

		// Determine whether valid packet received
		tlvType, tlvLength, err := util.DecodeTypeLength(recvBuf[:readSize])
		tlvSize := tlvType.Size() + tlvLength.Size() + int(tlvLength)
		if err == nil && readSize == tlvSize {
			// Packet was successfully received, send up to link service
			t.linkService.handleIncomingFrame(recvBuf[:tlvSize])
		}
	}

	t.changeState(Down)
}

func (t *UnicastUDPTransport) onClose() {
	core.LogInfo(t, "Closing UDP socket")
	t.hasQuit <- true
	unix.Close(t.socket)
}
