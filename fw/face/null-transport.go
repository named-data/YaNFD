/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// NullTransport is a transport that drops all packets.
type NullTransport struct {
	transportBase
}

// MakeNullTransport makes a NullTransport.
func MakeNullTransport() *NullTransport {
	var t NullTransport
	t.makeTransportBase(ndn.MakeNullFaceURI(), ndn.MakeNullFaceURI(), tlv.MaxNDNPacketSize)
	return &t
}
