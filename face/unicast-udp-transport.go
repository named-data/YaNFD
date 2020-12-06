/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"net"
	"syscall"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/util"
	"golang.org/x/sys/unix"
)

// UnicastUDPTransport is a unicast UDP transport
type UnicastUDPTransport struct {
	socket int
	transportBase
}

// MakeUnicastUDPTransport creates a new unicast UDP transport
func MakeUnicastUDPTransport(remoteURI URI, localURI URI) (*UnicastUDPTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || (remoteURI.Scheme() != "udp4" && remoteURI.Scheme() != "udp6") || !localURI.IsCanonical() || remoteURI.Scheme() != localURI.Scheme() {
		return nil, core.ErrNotCanonical
	}

	var t UnicastUDPTransport
	t.makeTransportBase(remoteURI, localURI, core.MaxNDNPacketSize)

	// Attempt to connect to remote URI
	var err error
	if t.localURI.Scheme() == "udp4" {
		var remoteAddr syscall.SockaddrInet4
		copy(remoteAddr.Addr[0:3], net.ParseIP(remoteURI.Path()))
		remoteAddr.Port = int(remoteURI.Port())
		t.socket, err = syscall.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	} else if t.localURI.Scheme() == "udp6" {
		var remoteAddr syscall.SockaddrInet6
		copy(remoteAddr.Addr[0:15], net.ParseIP(remoteURI.Path()))
		remoteAddr.Port = int(remoteURI.Port())
		t.socket, err = syscall.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	}
	if err != nil {
		return nil, err
	}

	err = syscall.SetsockoptInt(t.socket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		core.LogError(t, "Unable to allow address reuse:", err)
		return nil, err
	}

	if t.localURI.Scheme() == "udp4" {
		var localAddr syscall.SockaddrInet4
		copy(localAddr.Addr[0:3], net.ParseIP(localURI.Path()))
		localAddr.Port = int(localURI.Port())
		err = syscall.Bind(t.socket, &localAddr)
	} else if t.localURI.Scheme() == "udp6" {
		var localAddr syscall.SockaddrInet6
		copy(localAddr.Addr[0:15], net.ParseIP(localURI.Path()))
		localAddr.Port = int(localURI.Port())
		err = syscall.Bind(t.socket, &localAddr)
	}

	return &t, nil
}

func (t *UnicastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	for nBytesToWrite := len(frame); nBytesToWrite > 0; {
		nBytesWritten, err := syscall.Write(t.socket, frame[len(frame)-nBytesToWrite:])
		if err != nil {
			core.LogWarn("Unable to write on socket - DROP and Face DOWN")
			t.changeState(Down)
			t.hasQuit <- true
			return
		}
		nBytesToWrite -= nBytesWritten
	}
}

func (t *UnicastUDPTransport) runReceive() {
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit && t.state != Down {
		readSize, err := syscall.Read(t.socket, recvBuf)
		if err != nil {
			core.LogWarn(t, "Unable to read from socket (", err, ") - DROP and Face DOWN")
			t.changeState(Down)
			break
		}

		core.LogTrace(t, "Receive of size", readSize)

		if readSize > core.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP and Face DOWN")
			t.changeState(Down)
			break
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
	syscall.Close(t.socket)
}
