/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"sync"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/dispatch"
	"github.com/eric135/YaNFD/ndn"
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
	faceID := uint64(0)
	isExistingFaceID := true
	for isExistingFaceID {
		faceID = t.nextFaceID
		t.nextFaceID++
		_, isExistingFaceID = t.Faces[faceID]
	}
	t.Faces[faceID] = face
	face.SetFaceID(faceID)
	t.mutex.Unlock()

	// Add to dispatch
	dispatch.AddFace(faceID, face)

	core.LogDebug("FaceTable", "Registered FaceID=", faceID)
}

// Get gets the face with the specified ID (if any) from the face table.
func (t *Table) Get(id uint64) LinkService {
	t.mutex.RLock()
	face, ok := t.Faces[id]
	t.mutex.RUnlock()

	if ok {
		return face
	}
	return nil
}

// GetByURI gets the face with the specified remote URI (if any) from the face table.
func (t *Table) GetByURI(remoteURI *ndn.URI) LinkService {
	t.mutex.RLock()
	for _, face := range t.Faces {
		if face.RemoteURI().String() == remoteURI.String() {
			t.mutex.RUnlock()
			return face
		}
	}
	t.mutex.RUnlock()
	return nil
}

// GetAll returns points to all faces.
func (t *Table) GetAll() []LinkService {
	t.mutex.RLock()
	faces := make([]LinkService, 0, len(t.Faces))
	for _, face := range t.Faces {
		faces = append(faces, face)
	}
	t.mutex.RUnlock()
	return faces
}

// Remove removes a face from the face table.
func (t *Table) Remove(id uint64) {
	t.mutex.Lock()
	delete(t.Faces, id)
	t.mutex.Unlock()

	// Remove from dispatch
	dispatch.RemoveFace(id)

	core.LogDebug("FaceTable", "Unregistered FaceID=", id)
}
