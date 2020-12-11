// +build !windows
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

// UDPListener listens for incoming UDP unicast connections.
type UDPListener struct {
	socket   int
	localURI URI
	HasQuit  chan bool
}

// MakeUDPListener constructs a UDPListener.
func MakeUDPListener(localURI URI) (*UDPListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	var u UDPListener
	u.localURI = localURI
	u.HasQuit = make(chan bool, 1)
	return &u, nil
}

func (u *UDPListener) String() string {
	return "UDPListener, " + u.localURI.String()
}

// Run starts the UDP listener.
func (u *UDPListener) Run() {
	// Create listener
	var err error
	if u.localURI.Scheme() == "udp4" {
		u.socket, err = syscall.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	} else if u.localURI.Scheme() == "udp6" {
		u.socket, err = syscall.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	}
	if err != nil {
		core.LogError(u, "Unable to start UDP listener:", err)
		u.HasQuit <- true
		return
	}

	err = syscall.SetsockoptInt(u.socket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		core.LogError(u, "Unable to allow address reuse:", err)
		u.HasQuit <- true
		return
	}

	if u.localURI.Scheme() == "udp4" {
		var listenAddr syscall.SockaddrInet4
		copy(listenAddr.Addr[0:3], net.ParseIP(u.localURI.Path()))
		listenAddr.Port = 6364 // TODO
		err = syscall.Bind(u.socket, &listenAddr)
	} else if u.localURI.Scheme() == "udp6" {
		var listenAddr syscall.SockaddrInet6
		copy(listenAddr.Addr[0:15], net.ParseIP(u.localURI.Path()))
		listenAddr.Port = NDNUnicastUDPPort
		err = syscall.Bind(u.socket, &listenAddr)
	}

	if err != nil {
		core.LogError(u, "Unable to start UDP listener:", err)
		syscall.Close(u.socket)
		u.HasQuit <- true
		return
	}

	// Run accept loop
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit {
		readSize, remoteAddr, err := syscall.Recvfrom(u.socket, recvBuf, 0)
		if err != nil {
			core.LogWarn(u, "Unable to read from socket (", err, ") - DROP ")
			break
		}

		// Construct remote URI
		var remoteURI URI
		switch t := remoteAddr.(type) {
		case *syscall.SockaddrInet4:
			remoteURI = MakeUDPFaceURI(4, net.IP(t.Addr[0:]).String(), uint16(t.Port))
		case *syscall.SockaddrInet6:
			remoteURI = MakeUDPFaceURI(4, net.IP(t.Addr[0:]).String(), uint16(t.Port))
		}
		remoteURI.Canonize()
		if !remoteURI.IsCanonical() {
			core.LogWarn(u, "Unable to create face from", remoteURI.String(), " as remote URI is not canonical")
			continue
		}

		core.LogTrace(u, "Receive of size", readSize, "from", remoteURI.String())

		if readSize > core.MaxNDNPacketSize {
			core.LogWarn(u, "Received too much data without valid TLV block - DROP")
		}

		// Determine whether valid packet received
		tlvType, tlvLength, err := util.DecodeTypeLength(recvBuf[:readSize])
		tlvSize := tlvType.Size() + tlvLength.Size() + int(tlvLength)
		if err == nil && readSize == tlvSize {
			// If frame received here, must be for new remote endpoint
			newTransport, err := MakeUnicastUDPTransport(remoteURI, u.localURI)
			if err != nil {
				core.LogError(u, "Failed to create new unicast UDP transport:", err)
				continue
			}
			newLinkService := MakeNDNLPLinkService(newTransport)
			if err != nil {
				core.LogError(u, "Failed to create new NDNLPv2 transport:", err)
				continue
			}
			// Pass this frame to the link service for processing
			newLinkService.handleIncomingFrame(recvBuf[:tlvSize])

			// Add face to table and start its thread
			FaceTable.Add(newLinkService)
			newLinkService.Run()
		} else {
			core.LogDebug(u, "Received non-TLV from", remoteAddr)
		}
	}

	syscall.Close(u.socket)
	u.HasQuit <- true
}
