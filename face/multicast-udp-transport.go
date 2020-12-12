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

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/util"
	"golang.org/x/sys/unix"
)

// MulticastUDPTransport is a multicast UDP transport.
type MulticastUDPTransport struct {
	sendSocket int
	recvSocket int
	groupAddr  unix.Sockaddr
	localAddr  unix.Sockaddr
	isIPv4     bool
	transportBase
}

// MakeMulticastUDPTransport creates a new multicast UDP transport.
func MakeMulticastUDPTransport(localURI URI) (*MulticastUDPTransport, error) {
	// Validate local URI
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	var t MulticastUDPTransport
	t.isIPv4 = localURI.Scheme() == "udp4"

	// Get local interface
	localIf, err := InterfaceByIP(net.ParseIP(localURI.PathHost()))
	if err != nil || localIf == nil {
		core.LogError(t, "Unable to get interface for local URI", localURI.String(), ":", err)
	}

	// Format group and local addresses
	if t.isIPv4 {
		t.groupAddr = new(unix.SockaddrInet4)
		copy(t.groupAddr.(*unix.SockaddrInet4).Addr[0:3], net.ParseIP(t.remoteURI.Path()))
		t.groupAddr.(*unix.SockaddrInet4).Port = int(t.remoteURI.Port())

		t.localAddr = new(unix.SockaddrInet4)
		copy(t.localAddr.(*unix.SockaddrInet4).Addr[0:3], net.ParseIP(localURI.Path()))
		t.localAddr.(*unix.SockaddrInet4).Port = int(localURI.Port())
	} else {
		t.groupAddr = new(unix.SockaddrInet6)
		copy(t.groupAddr.(*unix.SockaddrInet6).Addr[0:15], net.ParseIP(t.remoteURI.PathHost()))
		t.groupAddr.(*unix.SockaddrInet6).Port = int(t.remoteURI.Port())

		t.localAddr = new(unix.SockaddrInet6)
		copy(t.localAddr.(*unix.SockaddrInet6).Addr[0:15], net.ParseIP(localURI.PathHost()))
		t.localAddr.(*unix.SockaddrInet6).Port = int(localURI.Port())
	}

	// Create send socket.
	if t.isIPv4 {
		t.makeTransportBase(DecodeURIString(NDNMulticastUDP4URI), localURI, core.MaxNDNPacketSize)
		// TODO: Get URI from config

		copy(t.localAddr.(*unix.SockaddrInet4).Addr[0:3], net.ParseIP(t.localURI.Path()))
		t.localAddr.(*unix.SockaddrInet4).Port = int(t.localURI.Port())
		t.sendSocket, err = unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
		if err != nil {
			return nil, err
		}

		err = unix.SetsockoptInt(t.sendSocket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			core.LogError(t, "Unable to allow address reuse:", err)
			unix.Close(t.sendSocket)
			return nil, err
		}

		err = unix.Bind(t.sendSocket, t.localAddr.(*unix.SockaddrInet4))
		if err != nil {
			unix.Close(t.sendSocket)
			return nil, err
		}
	} else {
		t.makeTransportBase(DecodeURIString(NDNMulticastUDP6URI), localURI, core.MaxNDNPacketSize)

		copy(t.localAddr.(*unix.SockaddrInet6).Addr[0:15], net.ParseIP(t.localURI.Path()))
		t.localAddr.(*unix.SockaddrInet6).Port = int(t.localURI.Port())
		t.sendSocket, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
		if err != nil {
			return nil, err
		}

		err = unix.SetsockoptInt(t.sendSocket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			core.LogError(t, "Unable to allow address reuse:", err)
			unix.Close(t.sendSocket)
			return nil, err
		}

		err = unix.Bind(t.sendSocket, t.localAddr.(*unix.SockaddrInet6))
		if err != nil {
			unix.Close(t.sendSocket)
			return nil, err
		}

		// Set scope
		t.scope = NonLocal
	}

	// Create receive socket
	if t.isIPv4 {
		t.recvSocket, err = unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
		if err != nil {
			unix.Close(t.sendSocket)
			return nil, err
		}

		err = unix.SetsockoptInt(t.recvSocket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			core.LogError(t, "Unable to allow address reuse:", err)
			unix.Close(t.sendSocket)
			unix.Close(t.recvSocket)
			return nil, err
		}

		var membershipRequest unix.IPMreqn
		copy(membershipRequest.Multiaddr[0:3], t.groupAddr.(*unix.SockaddrInet4).Addr[0:3])
		copy(membershipRequest.Address[0:3], t.localAddr.(*unix.SockaddrInet4).Addr[0:3])
		membershipRequest.Ifindex = int32(localIf.Index)
		err = unix.SetsockoptIPMreqn(t.recvSocket, unix.IPPROTO_IP, unix.IP_ADD_MEMBERSHIP, &membershipRequest)
		if err != nil {
			core.LogError(t, "Unable to add membership in IPv4 multicast group:", err)
			unix.Close(t.sendSocket)
			unix.Close(t.recvSocket)
			return nil, err
		}

		err = unix.Bind(t.recvSocket, t.localAddr.(*unix.SockaddrInet4))
		if err != nil {
			unix.Close(t.sendSocket)
			unix.Close(t.recvSocket)
			return nil, err
		}
	} else {
		t.recvSocket, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
		if err != nil {
			unix.Close(t.sendSocket)
			return nil, err
		}

		err = unix.SetsockoptInt(t.recvSocket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			core.LogError(t, "Unable to allow address reuse:", err)
			unix.Close(t.sendSocket)
			unix.Close(t.recvSocket)
			return nil, err
		}

		var membershipRequest unix.IPv6Mreq
		copy(membershipRequest.Multiaddr[0:15], t.groupAddr.(*unix.SockaddrInet6).Addr[0:15])
		membershipRequest.Interface = uint32(localIf.Index)
		err = unix.SetsockoptIPv6Mreq(t.recvSocket, unix.IPPROTO_IPV6, unix.IPV6_JOIN_GROUP, &membershipRequest)
		if err != nil {
			core.LogError(t, "Unable to join IPv6 multicast group:", err)
			unix.Close(t.sendSocket)
			unix.Close(t.recvSocket)
			return nil, err
		}

		err = unix.Bind(t.recvSocket, t.localAddr.(*unix.SockaddrInet6))
		if err != nil {
			unix.Close(t.sendSocket)
			unix.Close(t.recvSocket)
			return nil, err
		}
	}

	return &t, nil
}

func (t *MulticastUDPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size", len(frame))
	err := unix.Sendto(t.sendSocket, frame, 0, t.groupAddr)
	if err != nil {
		core.LogWarn("Unable to send on socket - DROP and Face DOWN")
		t.changeState(Down)
	}
}

func (t *MulticastUDPTransport) runReceive() {
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit && t.state != Down {
		readSize, remoteAddr, err := unix.Recvfrom(t.recvSocket, recvBuf, 0)
		if err != nil {
			core.LogWarn(t, "Unable to read from socket (", err, ") - DROP and Face DOWN")
			t.changeState(Down)
			break
		}

		switch r := remoteAddr.(type) {
		case *unix.SockaddrInet4:
			core.LogTrace(t, "Receive of size", readSize, "from", MakeUDPFaceURI(4, net.IP(r.Addr[0:]).String(), uint16(r.Port)))
		case *unix.SockaddrInet6:
			core.LogTrace(t, "Receive of size", readSize, "from", MakeUDPFaceURI(6, net.IP(r.Addr[0:]).String(), uint16(r.Port)))
		default:
			core.LogError(t, "Receive of size", readSize, "from unknown remote address type")
		}

		if readSize > core.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
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

func (t *MulticastUDPTransport) onClose() {
	core.LogInfo(t, "Closing UDP socket")
	t.hasQuit <- true
	unix.Close(t.sendSocket)
	unix.Close(t.recvSocket)
}
