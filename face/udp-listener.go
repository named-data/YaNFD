/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"context"
	"net"
	"strconv"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face/impl"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
)

// UDPListener listens for incoming UDP unicast connections.
type UDPListener struct {
	conn     net.PacketConn
	localURI *ndn_defn.URI
	HasQuit  chan bool
}

// MakeUDPListener constructs a UDPListener.
func MakeUDPListener(localURI *ndn_defn.URI) (*UDPListener, error) {
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
		core.LogError(l, "Unable to start UDP listener: ", err)
		l.HasQuit <- true
		return
	}

	// Run accept loop
	recvBuf := make([]byte, ndn_defn.MaxNDNPacketSize)
	for !core.ShouldQuit {
		readSize, remoteAddr, err := l.conn.ReadFrom(recvBuf)
		if err != nil {
			core.LogWarn(l, "Unable to read from socket (", err, ") - DROP ")
			break
		}

		// Construct remote URI
		var remoteURI *ndn_defn.URI
		host, port, err := net.SplitHostPort(remoteAddr.String())
		if err != nil {
			core.LogWarn(l, "Unable to create face from ", remoteAddr, ": could not split host from port")
			continue
		}
		portInt, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			core.LogWarn(l, "Unable to create face from ", remoteAddr, ": could not split host from port")
			continue
		}
		remoteURI = ndn_defn.MakeUDPFaceURI(4, host, uint16(portInt))
		remoteURI.Canonize()
		if !remoteURI.IsCanonical() {
			core.LogWarn(l, "Unable to create face from ", remoteURI, ": remote URI is not canonical")
			continue
		}

		core.LogTrace(l, "Receive of size ", readSize, " from ", remoteURI)

		// If frame received here, must be for new remote endpoint
		newTransport, err := MakeUnicastUDPTransport(remoteURI, l.localURI, PersistencyOnDemand)
		if err != nil {
			core.LogError(l, "Failed to create new unicast UDP transport: ", err)
			continue
		}
		newLinkService := MakeNDNLPLinkService(newTransport, MakeNDNLPLinkServiceOptions())

		// Add face to table (which assigns FaceID) before passing current frame to link service
		FaceTable.Add(newLinkService)
		go newLinkService.Run(recvBuf[:readSize])
	}

	l.conn.Close()
	l.HasQuit <- true
}
