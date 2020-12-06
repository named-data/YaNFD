/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import "github.com/eric135/YaNFD/core"

// NullTransport is a transport that drops all packets.
type NullTransport struct {
	transportBase
}

// MakeNullTransport makes a NullTransport.
func MakeNullTransport() *NullTransport {
	var t NullTransport
	t.makeTransportBase(MakeNullFaceURI(), MakeNullFaceURI(), core.MaxNDNPacketSize)
	return &t
}
