/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"errors"
	"net"
	"os"
	"path"

	"github.com/named-data/YaNFD/core"
	defn "github.com/named-data/YaNFD/defn"
)

// UnixStreamListener listens for incoming Unix stream connections.
type UnixStreamListener struct {
	conn     net.Listener
	localURI *defn.URI
	nextFD   int // We can't (at least easily) access the actual FD through net.Conn, so we'll make our own
	stopped  chan bool
}

// MakeUnixStreamListener constructs a UnixStreamListener.
func MakeUnixStreamListener(localURI *defn.URI) (*UnixStreamListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	return &UnixStreamListener{
		localURI: localURI,
		nextFD:   1,
		stopped:  make(chan bool, 1),
	}, nil
}

func (l *UnixStreamListener) String() string {
	return "UnixStreamListener, " + l.localURI.String()
}

func (l *UnixStreamListener) Run() {
	defer func() { l.stopped <- true }()

	// Delete any existing socket
	os.Remove(l.localURI.Path())

	// Create inside folder if not existing
	sockPath := l.localURI.Path()
	dirPath := path.Dir(sockPath)
	os.MkdirAll(dirPath, os.ModePerm)

	// Create listener
	var err error
	if l.conn, err = net.Listen(l.localURI.Scheme(), sockPath); err != nil {
		core.LogFatal(l, "Unable to start Unix stream listener: ", err)
	}

	// Set permissions to allow all local apps to communicate with us
	if err := os.Chmod(sockPath, os.ModePerm); err != nil {
		core.LogFatal(l, "Unable to change permissions on Unix stream listener: ", err)
	}

	core.LogInfo(l, "Listening")

	// Run accept loop
	for !core.ShouldQuit {
		newConn, err := l.conn.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			core.LogWarn(l, "Unable to accept connection: ", err)
			return
		}

		remoteURI := defn.MakeFDFaceURI(l.nextFD)
		l.nextFD++
		if !remoteURI.IsCanonical() {
			core.LogWarn(l, "Unable to create face from ", remoteURI, " as remote URI is not canonical")
			continue
		}

		newTransport, err := MakeUnixStreamTransport(remoteURI, l.localURI, newConn)
		if err != nil {
			core.LogError(l, "Failed to create new Unix stream transport: ", err)
			continue
		}

		core.LogInfo(l, "Accepting new Unix stream face ", remoteURI)
		MakeNDNLPLinkService(newTransport, MakeNDNLPLinkServiceOptions()).Run(nil)
	}
}

func (l *UnixStreamListener) Close() {
	if l.conn != nil {
		l.conn.Close()
		<-l.stopped
	}
}
