/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package dispatch

import (
	"sync"
)

// FaceDispatch is used to allow forwarding to interact with faces without a circular dependency issue.
var FaceDispatch map[int]Face

// FaceDispatchSync controls access to FaceDispatch.
var FaceDispatchSync sync.RWMutex

// FWDispatch is used to allow faces to interact with forwarding without a circular dependency issue.
var FWDispatch map[int]FWThread

// FWDispatchSync controls access to FWDispatch.
var FWDispatchSync sync.RWMutex

func init() {
	FaceDispatch = make(map[int]Face)
}

// AddFace adds the specified face to the dispatch list.
func AddFace(id int, face Face) {
	FaceDispatchSync.Lock()
	FaceDispatch[id] = face
	FaceDispatchSync.Unlock()
}

// GetFace returns the specified face or nil if it does not exist.
func GetFace(id int) Face {
	FaceDispatchSync.RLock()
	face, ok := FaceDispatch[id]
	FaceDispatchSync.RUnlock()
	if !ok {
		return nil
	}
	return face
}

// AddFWThread adds the specified forwarding thread to the dispatch list.
func AddFWThread(id int, thread FWThread) {
	FWDispatchSync.Lock()
	FWDispatch[id] = thread
	FWDispatchSync.Unlock()
}

// GetFWThread returns the specified forwarding thread or nil if it does not exist.
func GetFWThread(id int) FWThread {
	FWDispatchSync.RLock()
	thread, ok := FWDispatch[id]
	FWDispatchSync.RUnlock()
	if !ok {
		return nil
	}
	return thread
}
