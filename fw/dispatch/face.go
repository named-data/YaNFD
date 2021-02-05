/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package dispatch

import "github.com/eric135/YaNFD/ndn"

// Face provides an interface that faces can satisfy (to avoid circular dependency between faces and forwarding)
type Face interface {
	String() string
	SetFaceID(faceID int)

	FaceID() int
	LocalURI() *ndn.URI
	RemoteURI() *ndn.URI
	Scope() ndn.Scope
	MTU() int

	State() ndn.State

	SendPacket(packet *ndn.PendingPacket)
}
