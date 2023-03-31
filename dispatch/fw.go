/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package dispatch

import (
	"github.com/named-data/YaNFD/ndn"
)

// FWThread provides an interface that forwarding threads can satisfy
// (to avoid circular dependency between faces and forwarding)
type FWThread interface {
	String() string

	QueueData(packet *ndn.PendingPacket)
	QueueInterest(packet *ndn.PendingPacket)

	GetNumPitEntries() int
	GetNumCsEntries() int
}

// FWDispatch is used to allow faces to interact with forwarding without a circular dependency issue.
var FWDispatch []FWThread

// InitializeFWThreads sets up the forwarding thread dispatch slice.
func InitializeFWThreads(faces []FWThread) {
	FWDispatch = make([]FWThread, len(faces))
	copy(FWDispatch, faces)
}

// GetFWThread returns the specified forwarding thread or nil if it does not exist.
func GetFWThread(id int) FWThread {
	if id < 0 || id > len(FWDispatch) {
		return nil
	}
	return FWDispatch[id]
}
