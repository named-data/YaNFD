/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"context"
	"net"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face/impl"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// UDPListener listens for incoming UDP unicast connections.
type UDPListener struct {
	conn     net.PacketConn
	localURI URI
	HasQuit  chan bool
}

// MakeUDPListener constructs a UDPListener.
func MakeUDPListener(localURI URI) (*UDPListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	l := new(UDPListener)
	l.localURI = localURI
	l.HasQuit = make(chan bool, 1)
	return l, nil
}

func (l *UDPListener) String() string {
	return "UDPListener, " + l.localURI.String()
}

// Run starts the UDP listener.
func (l *UDPListener) Run() {
	// Create dialer and set reuse address option
	listenConfig := &net.ListenConfig{Control: impl.SyscallReuseAddr}

	// Create listener
	var err error
	var remote string
	if l.localURI.Scheme() == "udp4" {
		remote = l.localURI.PathHost() + ":" + strconv.Itoa(int(l.localURI.Port()))
	} else {
		remote = "[" + l.localURI.Path() + "]:" + strconv.Itoa(int(l.localURI.Port()))
	}
	l.conn, err = listenConfig.ListenPacket(context.Background(), l.localURI.Scheme(), remote)
	if err != nil {
		core.LogError(l, "Unable to start UDP listener:", err)
		l.HasQuit <- true
		return
	}

	// Run accept loop
	recvBuf := make([]byte, core.MaxNDNPacketSize)
	for !core.ShouldQuit {
		readSize, remoteAddr, err := l.conn.ReadFrom(recvBuf)
		if err != nil {
			core.LogWarn(l, "Unable to read from socket ("+err.Error()+") - DROP ")
			break
		}

		// Construct remote URI
		var remoteURI URI
		host, port, err := net.SplitHostPort(remoteAddr.String())
		if err != nil {
			core.LogWarn(l, "Unable to create face from "+remoteAddr.String()+": could not split host from port")
			continue
		}
		portInt, _ := strconv.ParseUint(port, 10, 16)
		if err != nil {
			core.LogWarn(l, "Unable to create face from "+remoteAddr.String()+": could not split host from port")
			continue
		}
		remoteURI = MakeUDPFaceURI(4, host, uint16(portInt))
		remoteURI.Canonize()
		if !remoteURI.IsCanonical() {
			core.LogWarn(l, "Unable to create face from "+remoteURI.String()+": remote URI is not canonical")
			continue
		}

		core.LogTrace(l, "Receive of size", readSize, "from", remoteURI.String())

		if readSize > core.MaxNDNPacketSize {
			core.LogWarn(l, "Received too much data without valid TLV block - DROP")
		}

		// Determine whether valid packet received
		_, _, tlvSize, err := tlv.DecodeTypeLength(recvBuf[:readSize])
		if err != nil {
			core.LogInfo("Unable to process received packet: " + err.Error())
		} else if readSize >= tlvSize {
			// If frame received here, must be for new remote endpoint
			newTransport, err := MakeUnicastUDPTransport(remoteURI, l.localURI)
			if err != nil {
				core.LogError(l, "Failed to create new unicast UDP transport:", err)
				continue
			}
			newLinkService := MakeNDNLPLinkService(newTransport)
			if err != nil {
				core.LogError(l, "Failed to create new NDNLPv2 transport:", err)
				continue
			}
			// Pass this frame to the link service for processing
			newLinkService.handleIncomingFrame(recvBuf[:tlvSize])

			// Add face to table and start its thread
			FaceTable.Add(newLinkService)
			newLinkService.Run()
		} else {
			core.LogDebug(l, "Received non-TLV from", remoteAddr)
		}
	}

	l.conn.Close()
	l.HasQuit <- true
}
