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
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/table"
)

// Thread Represents the management thread
type Thread struct {
	face      face.LinkService
	transport *face.InternalTransport
	prefix    *ndn.Name
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
	m.face, m.transport = face.RegisterInternalTransport()
	var err error
	m.prefix, err = ndn.NameFromString("/localhost/nfd")
	if err != nil {
		core.LogFatal(m, "Unable to create name for management prefix: "+err.Error())
	}
	table.FibStrategyTable.AddNexthop(m.prefix, m.face.FaceID(), 0)

	for {
		block, inFace := m.transport.Receive()
		if block == nil {
			// Indicates that internal face has quit, which means it's time for us to quit
			core.LogInfo(m, "Face quit, so management quitting")
			break
		}
		core.LogTrace(m, "Received block on face, IncomingFaceID="+strconv.Itoa(inFace))

		// We only expect Interests, so drop Data packets
		if block.Type() != tlv.Interest {
			core.LogWarn(m, "Dropping received non-Interest packet of type "+strconv.FormatUint(uint64(block.Type()), 10))
			continue
		}
		interest, err := ndn.DecodeInterest(block)
		if err != nil {
			core.LogWarn(m, "Unable to decode received Interest: "+err.Error()+" - DROP")
			continue
		}

		// Ensure Interest name matches expectations
		if interest.Name().Size() < m.prefix.Size()+2 { // Module + Verb
			core.LogInfo(m, "Control command name "+interest.Name().String()+" has unexpected number of components - DROP")
			continue
		}
		if !m.prefix.PrefixOf(interest.Name()) {
			core.LogInfo(m, "Control command name "+interest.Name().String()+" has unexpected prefix - DROP")
			continue
		}

		core.LogTrace(m, "Received management Interest "+interest.Name().String())

		// Dispatch interest based on name
		// TODO
	}
}
