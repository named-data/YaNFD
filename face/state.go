/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// State indicates the state of a face
type State int

const (
	// Up indicates the face is up
	Up State = iota
	// Down indicates the face is down
	Down State = iota
	// AdminDown indicates the face is administratively down
	AdminDown State = iota
)
