/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"strconv"
	"sync"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/dispatch"
)

// FaceTable is the global face table for this forwarder
var FaceTable Table

// Table hold all faces used by the forwarder.
type Table struct {
	Faces      map[int]LinkService
	mutex      sync.RWMutex
	nextFaceID int
}

func init() {
	FaceTable.Faces = make(map[int]LinkService)
	FaceTable.nextFaceID = 1
}

// Add adds a face to the face table.
func (t *Table) Add(face LinkService) {
	t.mutex.Lock()
	faceID := t.nextFaceID
	t.nextFaceID++
	t.Faces[faceID] = face
	face.SetFaceID(faceID)
	t.mutex.Unlock()

	// Add to dispatch
	dispatch.AddFace(faceID, face)

	core.LogDebug("FaceTable", "Registered "+strconv.Itoa(faceID))
}

// Get gets the face with the specified ID from the face table.
func (t *Table) Get(id int) LinkService {
	t.mutex.RLock()
	face, ok := t.Faces[id]
	t.mutex.RUnlock()

	if ok {
		return face
	}
	return nil
}

// Remove removes a face from the face table.
func (t *Table) Remove(id int) {
	t.mutex.Lock()
	delete(t.Faces, id)
	t.mutex.Unlock()

	// Remove from dispatch
	dispatch.FaceDispatchSync.Lock()
	delete(dispatch.FaceDispatch, id)
	dispatch.FaceDispatchSync.Unlock()

	core.LogDebug("FaceTable", "Unregistered "+strconv.Itoa(id))
}
