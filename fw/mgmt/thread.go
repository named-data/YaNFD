/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/mgmt"
	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/named-data/YaNFD/table"
)

// Thread Represents the management thread
type Thread struct {
	face           face.LinkService
	transport      *face.InternalTransport
	localPrefix    *ndn.Name
	nonLocalPrefix *ndn.Name
	modules        map[string]Module
}

// MakeMgmtThread creates a new management thread.
func MakeMgmtThread() *Thread {
	m := new(Thread)
	var err error
	m.localPrefix, err = ndn.NameFromString("/localhost/nfd")
	if err != nil {
		core.LogFatal(m, "Unable to create name for management prefix: ", err)
	}
	m.nonLocalPrefix, err = ndn.NameFromString("/localhop/nfd")
	if err != nil {
		core.LogFatal(m, "Unable to create name for management prefix: ", err)
	}
	m.modules = make(map[string]Module)
	m.registerModule("cs", new(ContentStoreModule))
	m.registerModule("faces", new(FaceModule))
	m.registerModule("fib", new(FIBModule))
	m.registerModule("rib", new(RIBModule))
	m.registerModule("status", new(ForwarderStatusModule))
	m.registerModule("strategy-choice", new(StrategyChoiceModule))
	return m
}

func (m *Thread) String() string {
	return "Management"
}

func (m *Thread) registerModule(name string, module Module) {
	m.modules[name] = module
	module.registerManager(m)
}

func (m *Thread) prefixLength() int {
	return m.localPrefix.Size()
}

func (m *Thread) sendResponse(response *mgmt.ControlResponse, interest *ndn.Interest, pitToken []byte, inFace uint64) {
	encodedResponse, err := response.Encode()
	if err != nil {
		core.LogWarn(m, "Unable to send ControlResponse for ", interest.Name(), ": ", err)
		return
	}
	encodedWire, err := encodedResponse.Wire()
	if err != nil {
		core.LogWarn(m, "Unable to send ControlResponse for ", interest.Name(), ": ", err)
		return
	}
	data := ndn.NewData(interest.Name(), encodedWire)

	encodedData, err := data.Encode()
	if err != nil {
		core.LogWarn(m, "Unable to send ControlResponse for ", interest.Name(), ": ", err)
		return
	}

	m.transport.Send(encodedData, pitToken, &inFace)
	core.LogTrace(m, "Sent ControlResponse for ", interest.Name())
}

// Run management thread
func (m *Thread) Run() {
	core.LogInfo(m, "Starting management")

	// Create and register Internal transport
	m.face, m.transport = face.RegisterInternalTransport()
	table.FibStrategyTable.InsertNextHop(m.localPrefix, m.face.FaceID(), 0)
	if enableLocalhopManagement {
		table.FibStrategyTable.InsertNextHop(m.nonLocalPrefix, m.face.FaceID(), 0)
	}

	for {
		block, pitToken, inFace := m.transport.Receive()
		if block == nil {
			// Indicates that internal face has quit, which means it's time for us to quit
			core.LogInfo(m, "Face quit, so management quitting")
			break
		}
		core.LogTrace(m, "Received block on face, IncomingFaceID=", inFace)

		// We only expect Interests, so drop Data packets
		if block.Type() != tlv.Interest {
			core.LogWarn(m, "Dropping received non-Interest packet of type ", block.Type())
			continue
		}
		interest, err := ndn.DecodeInterest(block)
		if err != nil {
			core.LogWarn(m, "Unable to decode received Interest: ", err, " - DROP")
			continue
		}

		// Ensure Interest name matches expectations
		if interest.Name().Size() < m.localPrefix.Size()+2 { // Module + Verb
			core.LogInfo(m, "Control command name ", interest.Name(), " has unexpected number of components - DROP")
			continue
		}
		if !m.localPrefix.PrefixOf(interest.Name()) && !m.nonLocalPrefix.PrefixOf(interest.Name()) {
			core.LogInfo(m, "Control command name ", interest.Name(), " has unexpected prefix - DROP")
			continue
		}

		core.LogTrace(m, "Received management Interest ", interest.Name())

		// Dispatch interest based on name
		moduleName := interest.Name().At(m.localPrefix.Size()).String()
		if module, ok := m.modules[moduleName]; ok {
			module.handleIncomingInterest(interest, pitToken, inFace)
		} else {
			core.LogWarn(m, "Received management Interest for unknown module ", moduleName)
			response := mgmt.MakeControlResponse(501, "Unknown module", nil)
			m.sendResponse(response, interest, pitToken, inFace)
		}
	}
}
