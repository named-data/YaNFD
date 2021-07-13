/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"github.com/named-data/YaNFD/table"
	"sync"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/ndn"
)

// FaceTable is the global face table for this forwarder
var FaceTable Table

// Table hold all faces used by the forwarder.
type Table struct {
	Faces      map[uint64]LinkService
	mutex      sync.RWMutex
	nextFaceID uint64
}

func init() {
	FaceTable.Faces = make(map[uint64]LinkService)
	FaceTable.nextFaceID = 1
}

// Add adds a face to the face table.
func (t *Table) Add(face LinkService) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	faceID := uint64(0)
	isExistingFaceID := true
	for isExistingFaceID {
		faceID = t.nextFaceID
		t.nextFaceID++
		_, isExistingFaceID = t.Faces[faceID]
	}
	t.Faces[faceID] = face
	face.SetFaceID(faceID)

	// Add to dispatch
	dispatch.AddFace(faceID, face)

	core.LogDebug("FaceTable", "Registered FaceID=", faceID)
}

// Get gets the face with the specified ID (if any) from the face table.
func (t *Table) Get(id uint64) LinkService {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	face, ok := t.Faces[id]

	if ok {
		return face
	}
	return nil
}

// GetByURI gets the face with the specified remote URI (if any) from the face table.
func (t *Table) GetByURI(remoteURI *ndn.URI) LinkService {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	for _, face := range t.Faces {
		if face.RemoteURI().String() == remoteURI.String() {

			return face
		}
	}
	return nil
}

// GetAll returns points to all faces.
func (t *Table) GetAll() []LinkService {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	faces := make([]LinkService, 0, len(t.Faces))
	for _, face := range t.Faces {
		faces = append(faces, face)
	}
	return faces
}

// Remove removes a face from the face table.
func (t *Table) Remove(id uint64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.Faces, id)

	// Remove from dispatch
	dispatch.RemoveFace(id)

	// Remove this face in RIB
	// Referential:
	// https://github.com/named-data/NFD/blob/7249fb4d5225cbe99a3901f9485a8ad99a7abceb/daemon/table/cleanup.cpp#L36-L40
	table.Rib.CleanUpFace(id)

	core.LogDebug("FaceTable", "Unregistered FaceID=", id)
}
