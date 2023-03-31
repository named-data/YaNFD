/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"errors"

	"github.com/named-data/YaNFD/ndn"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
)

const FaceEventsCacheSize = 100

// faceEvents caches face events. Note: should change to generic-typed list when available in Go 1.18.
var faceEvents [FaceEventsCacheSize]FaceEvent
var faceEventsIdx uint = 0
var faceEventsNextId uint64 = 0
var FaceEventSendFunc func(id uint64, pitToken []byte)

// faceEvent represents a face event for stream FaceEventNotification.
type FaceEvent struct {
	eventId       uint64
	faceEventKind FaceEventKind
	faceId        uint64
	remoteURI     *ndn.URI
	localURI      *ndn.URI
	scope         ndn.Scope
	persistency   Persistency
	linkType      ndn.LinkType
	flags         uint64
}

// FaceEventKind represents the type of a face event.
type FaceEventKind uint64

// Face event kinds.
const (
	FaceEventCreated   FaceEventKind = 1
	FaceEventDestroyed FaceEventKind = 2
	FaceEventUp        FaceEventKind = 3
	FaceEventDown      FaceEventKind = 4
)

func (p FaceEventKind) String() string {
	switch p {
	case FaceEventCreated:
		return "Created"
	case FaceEventDestroyed:
		return "Destroyed"
	case FaceEventUp:
		return "Up"
	default:
		return "Down"
	}
}

// EmitFaceEvent injects a new face event into the cache.
func EmitFaceEvent(kind FaceEventKind, face LinkService) {
	faceEvents[faceEventsIdx].eventId = faceEventsNextId
	faceEventsNextId++
	faceEvents[faceEventsIdx].faceEventKind = kind
	if face != nil {
		faceEvents[faceEventsIdx].faceId = face.FaceID()
		faceEvents[faceEventsIdx].remoteURI = face.RemoteURI()
		faceEvents[faceEventsIdx].localURI = face.LocalURI()
		faceEvents[faceEventsIdx].scope = face.Scope()
		faceEvents[faceEventsIdx].persistency = face.Persistency()
		faceEvents[faceEventsIdx].linkType = face.LinkType()
	}
	lp, ok := face.(*NDNLPLinkService)
	if ok {
		faceEvents[faceEventsIdx].flags = lp.options.Flags()
	} else {
		faceEvents[faceEventsIdx].flags = 0
	}
	oldId := faceEventsNextId - 1
	faceEventsIdx = (faceEventsIdx + 1) % FaceEventsCacheSize
	if FaceEventSendFunc != nil {
		FaceEventSendFunc(oldId, nil)
	}
}

// GetFaceEvent returns the face event with the given id.
// It will return nil if the specified event is discarded or does not exist.
func GetFaceEvent(eventId uint64) *FaceEvent {
	if eventId >= faceEventsNextId || eventId < faceEventsNextId-FaceEventsCacheSize {
		return nil
	}
	idx := (faceEventsIdx + uint(eventId+FaceEventsCacheSize-faceEventsNextId)) % FaceEventsCacheSize
	return &faceEvents[idx]
}

// FaceEventLastId return the id of the last face event.
// It will overflow if there is no face event, but this is safe and nearly impossible to happen.
func FaceEventLastId() uint64 {
	return faceEventsNextId - 1
}

// Encode encodes a FaceEventNotification.
func (f *FaceEvent) Encode() (enc.Wire, error) {
	if f.remoteURI == nil {
		return nil, errors.New("URI is required, but unset")
	}
	if f.localURI == nil {
		return nil, errors.New("LocalUri is required, but unset")
	}
	notif := &mgmt.FaceEventNotification{
		Val: &mgmt.FaceEventNotificationValue{
			FaceEventKind:   uint64(f.faceEventKind),
			FaceId:          f.faceId,
			Uri:             f.remoteURI.String(),
			LocalUri:        f.localURI.String(),
			FaceScope:       uint64(f.scope),
			FacePersistency: uint64(f.persistency),
			LinkType:        uint64(f.linkType),
			Flags:           f.flags,
		},
	}

	return notif.Encode(), nil
}
