/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// FaceTable is the global face table for this forwarder
var FaceTable Table

// Table hold all faces used by the forwarder.
type Table struct {
	Faces      map[int]LinkService
	nextFaceID int
}

// MakeTable creates and initializes the face table.
func MakeTable() Table {
	var t Table
	t.Faces = make(map[int]LinkService)
	t.nextFaceID = 1
	return t
}

// Add adds a face to the face table.
func (t *Table) Add(face LinkService) {
	t.Faces[t.nextFaceID] = face
	face.setFaceID(t.nextFaceID)
	t.nextFaceID++
}

// Remove removes a face from the face table.
func (t *Table) Remove(id int) {
	delete(t.Faces, id)
}
