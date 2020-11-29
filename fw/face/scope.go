/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// Scope indicates the scope of a face
type Scope int

const (
	// Local indicates the face is local (to an application)
	Local State = iota
	// NonLocal indicates the face is non-local (to another forwarder)
	NonLocal State = iota
)
