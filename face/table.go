/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"sync"
	"sync/atomic"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
	"github.com/named-data/YaNFD/table"
)

// FaceTable is the global face table for this forwarder
var FaceTable Table

// Table hold all faces used by the forwarder.
type Table struct {
	faces      sync.Map
	nextFaceID atomic.Uint64
}

func init() {
	FaceTable.faces = sync.Map{}
	FaceTable.nextFaceID.Store(1)
}

// Add adds a face to the face table.
func (t *Table) Add(face LinkService) {
	faceID := t.nextFaceID.Add(1) - 1
	face.SetFaceID(faceID)
	t.faces.Store(faceID, face)
	dispatch.AddFace(faceID, face)
	core.LogDebug("FaceTable", "Registered FaceID=", faceID)
}

// Get gets the face with the specified ID (if any) from the face table.
func (t *Table) Get(id uint64) LinkService {
	face, ok := t.faces.Load(id)

	if ok {
		return face.(LinkService)
	}
	return nil
}

// GetByURI gets the face with the specified remote URI (if any) from the face table.
func (t *Table) GetByURI(remoteURI *ndn_defn.URI) LinkService {
	var found LinkService
	t.faces.Range(func(_, face interface{}) bool {
		if face.(LinkService).RemoteURI().String() == remoteURI.String() {
			found = face.(LinkService)
			return false
		}
		return true
	})
	return found
}

// GetAll returns points to all faces.
func (t *Table) GetAll() []LinkService {
	faces := make([]LinkService, 0)
	t.faces.Range(func(_, face interface{}) bool {
		faces = append(faces, face.(LinkService))
		return true
	})
	return faces
}

// Remove removes a face from the face table.
func (t *Table) Remove(id uint64) {
	t.faces.Delete(id)
	dispatch.RemoveFace(id)
	table.Rib.CleanUpFace(id)
	core.LogDebug("FaceTable", "Unregistered FaceID=", id)
}
