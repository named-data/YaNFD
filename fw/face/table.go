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
	"time"

	"github.com/pulsejet/ndnd/fw/core"
	defn "github.com/pulsejet/ndnd/fw/defn"
	"github.com/pulsejet/ndnd/fw/dispatch"
	"github.com/pulsejet/ndnd/fw/table"
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
	go FaceTable.ExpirationHandler()
}

func (t *Table) String() string {
	return "FaceTable"
}

// Add adds a face to the face table.
func (t *Table) Add(face LinkService) {
	faceID := t.nextFaceID.Add(1) - 1
	face.SetFaceID(faceID)
	t.faces.Store(faceID, face)
	dispatch.AddFace(faceID, face)
	core.LogDebug(t, "Registered FaceID=", faceID)
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
func (t *Table) GetByURI(remoteURI *defn.URI) LinkService {
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
	core.LogInfo(t, "Unregistered FaceID=", id)
}

// ExpirationHandler stops the faces that have expired
func (t *Table) ExpirationHandler() {
	for {
		// Check for expired faces every 10 seconds
		time.Sleep(10 * time.Second)

		// Iterate the face table
		t.faces.Range(func(_, face interface{}) bool {
			transport := face.(LinkService).Transport()
			if transport != nil && transport.ExpirationPeriod() < 0 {
				core.LogInfo(transport, "Face expired")
				transport.Close()
			}
			return true
		})
	}
}
