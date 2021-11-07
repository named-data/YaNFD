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
	"github.com/named-data/YaNFD/ndn/tlv"
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
	faceEvents[faceEventsIdx].faceId = face.FaceID()
	faceEvents[faceEventsIdx].remoteURI = face.RemoteURI()
	faceEvents[faceEventsIdx].localURI = face.LocalURI()
	faceEvents[faceEventsIdx].scope = face.Scope()
	faceEvents[faceEventsIdx].persistency = face.Persistency()
	faceEvents[faceEventsIdx].linkType = face.LinkType()
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
	if eventId >= faceEventsNextId || eventId+FaceEventsCacheSize < faceEventsNextId {
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
func (f *FaceEvent) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.FaceEventNotification)

	wire.Append(tlv.EncodeNNIBlock(tlv.FaceEventKind, uint64(f.faceEventKind)))
	wire.Append(tlv.EncodeNNIBlock(tlv.FaceID, f.faceId))
	if f.remoteURI == nil {
		return nil, errors.New("URI is required, but unset")
	}
	wire.Append(tlv.NewBlock(tlv.URI, []byte(f.remoteURI.String())))
	if f.localURI == nil {
		return nil, errors.New("LocalUri is required, but unset")
	}
	wire.Append(tlv.NewBlock(tlv.LocalURI, []byte(f.localURI.String())))
	wire.Append(tlv.EncodeNNIBlock(tlv.FaceScope, uint64(f.scope)))
	wire.Append(tlv.EncodeNNIBlock(tlv.FacePersistency, uint64(f.persistency)))
	wire.Append(tlv.EncodeNNIBlock(tlv.LinkType, uint64(f.linkType)))
	wire.Append(tlv.EncodeNNIBlock(tlv.Flags, f.flags))

	wire.Encode()
	return wire, nil
}
