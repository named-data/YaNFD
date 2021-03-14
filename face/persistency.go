/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// Persistency represents the persistency of a face.
type Persistency uint64

// Face persistencies (shared with management).
const (
	PersistencyPersistent Persistency = 0
	PersistencyOnDemand   Persistency = 1
	PersistencyPermanent  Persistency = 2
)

func (p Persistency) String() string {
	switch p {
	case PersistencyPersistent:
		return "Persistent"
	case PersistencyOnDemand:
		return "OnDemand"
	case PersistencyPermanent:
		return "Permanent"
	default:
		return "Unknown"
	}
}
