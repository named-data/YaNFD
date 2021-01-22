/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package dispatch

import "github.com/eric135/YaNFD/ndn"

// FWThread provides an interface that forwarding threads can satisfy (to avoid circular dependency between faces and forwarding)
type FWThread interface {
	String() string

	QueueData(packet *ndn.PendingPacket)
	QueueInterest(packet *ndn.PendingPacket)
}
