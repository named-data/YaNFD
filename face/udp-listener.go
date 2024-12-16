/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/named-data/YaNFD/core"
	defn "github.com/named-data/YaNFD/defn"
	"github.com/named-data/YaNFD/face/impl"
)

// UDPListener listens for incoming UDP unicast connections.
type UDPListener struct {
	conn     net.PacketConn
	localURI *defn.URI
	stopped  chan bool
}

// MakeUDPListener constructs a UDPListener.
func MakeUDPListener(localURI *defn.URI) (*UDPListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "udp4" && localURI.Scheme() != "udp6") {
		return nil, core.ErrNotCanonical
	}

	l := new(UDPListener)
	l.localURI = localURI
	l.stopped = make(chan bool, 1)
	return l, nil
}

func (l *UDPListener) String() string {
	return fmt.Sprintf("UDPListener, %s", l.localURI)
}

// Run starts the UDP listener.
func (l *UDPListener) Run() {
	defer func() { l.stopped <- true }()

	// Create dialer and set reuse address option
	listenConfig := &net.ListenConfig{Control: impl.SyscallReuseAddr}

	// Create listener
	var remote string
	if l.localURI.Scheme() == "udp4" {
		remote = fmt.Sprintf("%s:%d", l.localURI.PathHost(), l.localURI.Port())
	} else {
		remote = fmt.Sprintf("[%s]:%d", l.localURI.Path(), l.localURI.Port())
	}

	// Start listening for incoming connections
	var err error
	l.conn, err = listenConfig.ListenPacket(context.Background(), l.localURI.Scheme(), remote)
	if err != nil {
		core.LogError(l, "Unable to start UDP listener: ", err)
		return
	}

	// Run accept loop
	recvBuf := make([]byte, defn.MaxNDNPacketSize)
	for !core.ShouldQuit {
		readSize, remoteAddr, err := l.conn.ReadFrom(recvBuf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			core.LogWarn(l, "Unable to read from socket (", err, ") - DROP ")
			return
		}

		// Construct remote URI
		var remoteURI *defn.URI
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
		remoteURI = defn.MakeUDPFaceURI(4, host, uint16(portInt))
		remoteURI.Canonize()
		if !remoteURI.IsCanonical() {
			core.LogWarn(l, "Unable to create face from ", remoteURI, ": remote URI is not canonical")
			continue
		}

		// If frame received here, must be for new remote endpoint
		newTransport, err := MakeUnicastUDPTransport(remoteURI, l.localURI, PersistencyOnDemand)
		if err != nil {
			core.LogError(l, "Failed to create new unicast UDP transport: ", err)
			continue
		}

		core.LogInfo(l, "Accepting new UDP face ", newTransport.RemoteURI())
		MakeNDNLPLinkService(newTransport, MakeNDNLPLinkServiceOptions()).Run(recvBuf[:readSize])
	}
}

func (l *UDPListener) Close() {
	if l.conn != nil {
		l.conn.Close()
		<-l.stopped
	}
}
