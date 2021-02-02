/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// NullTransport is a transport that drops all packets.
type NullTransport struct {
	transportBase
}

// MakeNullTransport makes a NullTransport.
func MakeNullTransport() *NullTransport {
	t := new(NullTransport)
	t.makeTransportBase(ndn.MakeNullFaceURI(), ndn.MakeNullFaceURI(), tlv.MaxNDNPacketSize)
	t.changeState(ndn.Up)
	return t
}

func (t *NullTransport) String() string {
	return "NullTransport, FaceID=" + strconv.Itoa(t.faceID) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

func (t *NullTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "- state:", t.state, "->", new)
	t.state = new

	if t.state != ndn.Up {
		// Stop link service
		t.linkService.tellTransportQuit()
	}
}
