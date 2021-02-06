/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

// LinkType indicates what type of link a face is.
type LinkType int

const (
	// PointToPoint is a face with one remote endpoint.
	PointToPoint LinkType = 0
	// MultiAccess is a face that communicates with a multicast group.
	MultiAccess LinkType = 1
	// AdHoc communicates on a wireless ad-hoc network.
	AdHoc LinkType = 2
)
