/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"errors"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// FaceStatus contains status information about a face.
type FaceStatus struct {
	FaceID                        uint64
	URI                           *ndn.URI
	LocalURI                      *ndn.URI
	ExpirationPeriod              *uint64
	FaceScope                     uint64
	FacePersistency               uint64
	LinkType                      uint64
	BaseCongestionMarkingInterval *uint64
	DefaultCongestionThreshold    *uint64
	MTU                           *uint64
	NInInterests                  uint64
	NInData                       uint64
	NInNacks                      uint64
	NOutInterests                 uint64
	NOutData                      uint64
	NOutNacks                     uint64
	NInBytes                      uint64
	NOutBytes                     uint64
	Flags                         uint64
}

// MakeFaceStatus creates an empty FaceStatus.
func MakeFaceStatus() *FaceStatus {
	f := new(FaceStatus)
	return f
}

// Encode encodes a FaceStatus.
func (f *FaceStatus) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.FaceStatus)

	wire.Append(tlv.EncodeNNIBlock(tlv.FaceID, f.FaceID))
	if f.URI == nil {
		return nil, errors.New("URI is required, but unset")
	}
	wire.Append(tlv.NewBlock(tlv.URI, []byte(f.URI.String())))
	if f.LocalURI == nil {
		return nil, errors.New("LocalURI is required, but unset")
	}
	wire.Append(tlv.NewBlock(tlv.LocalURI, []byte(f.LocalURI.String())))
	if f.ExpirationPeriod != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.ExpirationPeriod, *f.ExpirationPeriod))
	}
	wire.Append(tlv.EncodeNNIBlock(tlv.FaceScope, f.FaceScope))
	wire.Append(tlv.EncodeNNIBlock(tlv.FacePersistency, f.FacePersistency))
	wire.Append(tlv.EncodeNNIBlock(tlv.LinkType, f.LinkType))
	if f.BaseCongestionMarkingInterval != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.BaseCongestionMarkingInterval, *f.BaseCongestionMarkingInterval))
	}
	if f.DefaultCongestionThreshold != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.DefaultCongestionThreshold, *f.DefaultCongestionThreshold))
	}
	if f.MTU != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.MTU, *f.MTU))
	}
	wire.Append(tlv.EncodeNNIBlock(tlv.NInInterests, f.NInInterests))
	wire.Append(tlv.EncodeNNIBlock(tlv.NInData, f.NInData))
	wire.Append(tlv.EncodeNNIBlock(tlv.NInNacks, f.NInNacks))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutInterests, f.NOutInterests))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutData, f.NOutData))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutNacks, f.NOutNacks))
	wire.Append(tlv.EncodeNNIBlock(tlv.NInBytes, f.NInBytes))
	wire.Append(tlv.EncodeNNIBlock(tlv.NOutBytes, f.NOutBytes))
	wire.Append(tlv.EncodeNNIBlock(tlv.Flags, f.Flags))

	wire.Encode()
	return wire, nil
}
