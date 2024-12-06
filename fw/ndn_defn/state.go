/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_defn

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

func (s State) String() string {
	switch s {
	case Up:
		return "Up"
	case Down:
		return "Down"
	case AdminDown:
		return "AdminDown"
	default:
		return "Unknown"
	}
}
