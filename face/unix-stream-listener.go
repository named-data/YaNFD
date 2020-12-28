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
	"strconv"

	"github.com/eric135/YaNFD/core"
)

// UnixStreamListener listens for incoming Unix stream connections.
type UnixStreamListener struct {
	conn     net.Listener
	localURI URI
	HasQuit  chan bool
}

// MakeUnixStreamListener constructs a UnixStreamListener.
func MakeUnixStreamListener(localURI URI) (*UnixStreamListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	l := new(UnixStreamListener)
	l.localURI = localURI
	l.HasQuit = make(chan bool, 1)
	return l, nil
}

func (l *UnixStreamListener) String() string {
	return "UnixStreamListener, " + l.localURI.String()
}

// Run starts the Unix stream listener.
func (l *UnixStreamListener) Run() {
	// Create listener
	var err error
	l.conn, err = net.Listen(l.localURI.Scheme(), l.localURI.Path()+":"+strconv.Itoa(int(l.localURI.Port())))
	if err != nil {
		core.LogError(l, "Unable to start Unix stream listener:", err)
		l.HasQuit <- true
		return
	}

	// Run accept loop
	for !core.ShouldQuit {
		newConn, err := l.conn.Accept()
		if err != nil {
			core.LogWarn(l, "Unable to accept connection: "+err.Error())
			break
		}

		// Construct remote URI
		fd, err := strconv.Atoi(newConn.LocalAddr().(*net.UnixAddr).String())
		if err != nil {
			core.LogWarn(l, "Unable to parse FD: "+err.Error())
			continue
		}
		remoteURI := MakeFDFaceURI(fd)
		if !remoteURI.IsCanonical() {
			core.LogWarn(l, "Unable to create face from", remoteURI.String(), " as remote URI is not canonical")
			continue
		}

		newTransport, err := MakeUnixStreamTransport(remoteURI, l.localURI, newConn)
		if err != nil {
			core.LogError(l, "Failed to create new Unix stream transport:", err)
			continue
		}
		newLinkService := MakeNDNLPLinkService(newTransport)
		if err != nil {
			core.LogError(l, "Failed to create new NDNLPv2 transport:", err)
			continue
		}

		core.LogInfo(l, "Creating new Unix stream face", remoteURI)

		// Add face to table and start its thread
		FaceTable.Add(newLinkService)
		newLinkService.Run()
	}

	l.conn.Close()
	l.HasQuit <- true
}
