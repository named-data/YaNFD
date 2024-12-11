/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package defn

// Scope indicates the scope of a face
type Scope int

const (
	// Unknown indicates that the scope is unknown.
	Unknown Scope = -1
	// NonLocal indicates the face is non-local (to another forwarder).
	NonLocal Scope = 0
	// Local indicates the face is local (to an application).
	Local Scope = 1
)
