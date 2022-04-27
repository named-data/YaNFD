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
	"github.com/named-data/YaNFD/ndn"
)

// TCPListener listens for incoming TCP unicast connections.
type TCPListener struct {
	conn     net.Listener
	localURI *ndn.URI
	HasQuit  chan bool
}

// MakeTCPListener constructs a TCPListener.
func MakeTCPListener(localURI *ndn.URI) (*TCPListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || (localURI.Scheme() != "tcp4" && localURI.Scheme() != "tcp6") {
		return nil, core.ErrNotCanonical
	}

	l := new(TCPListener)
	l.localURI = localURI
	l.HasQuit = make(chan bool, 1)
	return l, nil
}

func (l *TCPListener) String() string {
	return "TCPListener, " + l.localURI.String()
}

// Run starts the TCP listener.
func (l *TCPListener) Run() {
	// Create dialer and set reuse address option
	listenConfig := &net.ListenConfig{Control: impl.SyscallReuseAddr}

	// Create listener
	var err error
	var remote string
	if l.localURI.Scheme() == "tcp4" {
		remote = l.localURI.PathHost() + ":" + strconv.Itoa(int(l.localURI.Port()))
	} else {
		remote = "[" + l.localURI.Path() + "]:" + strconv.Itoa(int(l.localURI.Port()))
	}
	l.conn, err = listenConfig.Listen(context.Background(), l.localURI.Scheme(), remote)
	if err != nil {
		core.LogError(l, "Unable to start TCP listener: ", err)
		l.HasQuit <- true
		return
	}

	// Run accept loop
	for !core.ShouldQuit {
		remoteConn, err := l.conn.Accept()
		if err != nil {
			core.LogWarn(l, "Unable to accept connection: ", err)
			continue
		}

		newTransport, err := AcceptUnicastTCPTransport(remoteConn, l.localURI, PersistencyPersistent)
		if err != nil {
			core.LogError(l, "Failed to create new unicast TCP transport: ", err)
			continue
		}
		newLinkService := MakeNDNLPLinkService(newTransport, MakeNDNLPLinkServiceOptions())

		// Add face to table (which assigns FaceID) before passing current frame to link service
		FaceTable.Add(newLinkService)
		go newLinkService.Run(nil)
	}

	l.HasQuit <- true
}

// Close closes the TCPListener.
func (l *TCPListener) Close() {
	core.LogInfo(l, "Stopping listener")
	if l.conn != nil {
		l.conn.Close()
		l.conn = nil
	}
}
