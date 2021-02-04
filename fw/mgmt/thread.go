/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face"
)

// Thread Represents the management thread
type Thread struct {
}

// MakeMgmtThread creates a new management thread.
func MakeMgmtThread() *Thread {
	m := new(Thread)
	return m
}

func (m *Thread) String() string {
	return "Management"
}

// Run management thread
func (m *Thread) Run() {
	core.LogInfo(m, "Starting management")

	// Create and register Internal transport
	transport := face.RegisterInternalTransport()

	for {
		block, inFace := transport.Receive()
		if block == nil {
			// Indicates that internal face has quit, which means it's time for us to quit
			core.LogInfo(m, "Face quit, so management quitting")
			break
		}
		core.LogTrace(m, "Received block on face, IncomingFaceID="+strconv.Itoa(inFace))
	}
}
