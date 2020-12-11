// +build !windows
/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"syscall"

	"github.com/eric135/YaNFD/core"
	"golang.org/x/sys/unix"
)

// UnixStreamListener listens for incoming Unix stream connections.
type UnixStreamListener struct {
	socket   int
	localURI URI
	HasQuit  chan bool
}

// MakeUnixStreamListener constructs a UnixStreamListener.
func MakeUnixStreamListener(localURI URI) (*UnixStreamListener, error) {
	localURI.Canonize()
	if !localURI.IsCanonical() || localURI.Scheme() != "unix" {
		return nil, core.ErrNotCanonical
	}

	var u UnixStreamListener
	u.localURI = localURI
	u.HasQuit = make(chan bool, 1)
	return &u, nil
}

func (u *UnixStreamListener) String() string {
	return "UnixStreamListener, " + u.localURI.String()
}

// Run starts the Unix stream listener.
func (u *UnixStreamListener) Run() {
	// Create listener
	var err error
	u.socket, err = syscall.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		core.LogError(u, "Unable to start Unix stream listener:", err)
		u.HasQuit <- true
		return
	}

	err = syscall.SetsockoptInt(u.socket, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		core.LogError(u, "Unable to allow address reuse:", err)
		u.HasQuit <- true
		return
	}

	listenAddr := syscall.SockaddrUnix{Name: u.localURI.Path()}
	err = syscall.Bind(u.socket, &listenAddr)
	if err != nil {
		core.LogError(u, "Unable to start UDP listener:", err)
		syscall.Close(u.socket)
		u.HasQuit <- true
		return
	}

	// Run accept loop
	for !core.ShouldQuit {
		fd, _, err := syscall.Accept(u.socket)
		if err != nil {
			core.LogWarn(u, "Unable to read from socket (", err, ") - DROP ")
			break
		}

		// Construct remote URI
		remoteURI := MakeFDFaceURI(fd)
		if !remoteURI.IsCanonical() {
			core.LogWarn(u, "Unable to create face from", remoteURI.String(), " as remote URI is not canonical")
			continue
		}

		newTransport, err := MakeUnixStreamTransport(remoteURI, u.localURI, fd)
		if err != nil {
			core.LogError(u, "Failed to create new Unix stream transport:", err)
			continue
		}
		newLinkService := MakeNDNLPLinkService(newTransport)
		if err != nil {
			core.LogError(u, "Failed to create new NDNLPv2 transport:", err)
			continue
		}

		core.LogInfo(u, "Creating new Unix stream face", remoteURI)

		// Add face to table and start its thread
		FaceTable.Add(newLinkService)
		newLinkService.Run()
	}

	syscall.Close(u.socket)
	u.HasQuit <- true
}
