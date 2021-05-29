/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package dispatch

import (
	"sync"

	"github.com/named-data/YaNFD/ndn"
)

// Face provides an interface that faces can satisfy (to avoid circular dependency between faces and forwarding)
type Face interface {
	String() string
	SetFaceID(faceID uint64)

	FaceID() uint64
	LocalURI() *ndn.URI
	RemoteURI() *ndn.URI
	Scope() ndn.Scope
	LinkType() ndn.LinkType
	MTU() int

	State() ndn.State

	SendPacket(packet *ndn.PendingPacket)
}

// FaceDispatch is used to allow forwarding to interact with faces without a circular dependency issue.
var FaceDispatch map[uint64]Face

// FaceDispatchSync controls access to FaceDispatch.
var FaceDispatchSync sync.RWMutex

func init() {
	FaceDispatch = make(map[uint64]Face)
}

// AddFace adds the specified face to the dispatch list.
func AddFace(id uint64, face Face) {
	FaceDispatchSync.Lock()
	FaceDispatch[id] = face
	FaceDispatchSync.Unlock()
}

// GetFace returns the specified face or nil if it does not exist.
func GetFace(id uint64) Face {
	FaceDispatchSync.RLock()
	face, ok := FaceDispatch[id]
	FaceDispatchSync.RUnlock()
	if !ok {
		return nil
	}
	return face
}

// RemoveFace removes the specified face from the dispatch map.
func RemoveFace(id uint64) {
	FaceDispatchSync.Lock()
	delete(FaceDispatch, id)
	FaceDispatchSync.Unlock()
}
