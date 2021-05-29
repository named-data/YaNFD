/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"errors"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/tlv"
)

// FaceQueryFilter is a filter used to retrieve a subset of faces matching the filter.
type FaceQueryFilter struct {
	FaceID          *uint64
	URIScheme       *string
	URI             *ndn.URI
	LocalURI        *ndn.URI
	FaceScope       *uint64
	FacePersistency *uint64
	LinkType        *uint64
}

// MakeFaceQueryFilter creates an empty FaceQueryFilter.
func MakeFaceQueryFilter() *FaceQueryFilter {
	f := new(FaceQueryFilter)
	return f
}

// DecodeFaceQueryFilterFromEncoded decodes a FaceQueryFilter from an encoded byte string.
func DecodeFaceQueryFilterFromEncoded(wire []byte) (*FaceQueryFilter, error) {
	block, _, err := tlv.DecodeBlock(wire)
	if err != nil {
		return nil, err
	}
	return DecodeFaceQueryFilter(block)
}

// DecodeFaceQueryFilter decodes a FaceQueryFilter from the wire.
func DecodeFaceQueryFilter(wire *tlv.Block) (*FaceQueryFilter, error) {
	if wire == nil {
		return nil, errors.New("wire is unset")
	}

	if wire.Type() != tlv.FaceQueryFilter {
		return nil, tlv.ErrUnexpected
	}

	c := new(FaceQueryFilter)

	wire.Parse()
	var err error
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case tlv.FaceID:
			if c.FaceID != nil {
				return nil, errors.New("duplicate FaceId")
			}
			c.FaceID = new(uint64)
			*c.FaceID, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("unable to decode FaceId: " + err.Error())
			}
		case tlv.URIScheme:
			if c.URIScheme != nil {
				return nil, errors.New("duplicate UriScheme")
			}
			c.URIScheme = new(string)
			*c.URIScheme = string(elem.Value())
		case tlv.URI:
			if c.URI != nil {
				return nil, errors.New("duplicate Uri")
			}
			c.URI = ndn.DecodeURIString(string(elem.Value()))
			if err != nil {
				return nil, errors.New("unable to decode Uri: " + err.Error())
			}
		case tlv.LocalURI:
			if c.LocalURI != nil {
				return nil, errors.New("duplicate LocalUri")
			}
			c.LocalURI = ndn.DecodeURIString(string(elem.Value()))
			if err != nil {
				return nil, errors.New("unable to decode LocalUri: " + err.Error())
			}
		case tlv.FaceScope:
			if c.FaceScope != nil {
				return nil, errors.New("duplicate FaceScope")
			}
			c.FaceScope = new(uint64)
			*c.FaceScope, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("unable to decode FaceScope: " + err.Error())
			}
		case tlv.FacePersistency:
			if c.FacePersistency != nil {
				return nil, errors.New("duplicate FacePersistency")
			}
			c.FacePersistency = new(uint64)
			*c.FacePersistency, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("unable to decode FacePersistency: " + err.Error())
			}
		case tlv.LinkType:
			if c.LinkType != nil {
				return nil, errors.New("duplicate LinkType")
			}
			c.LinkType = new(uint64)
			*c.LinkType, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("unable to decode LinkType: " + err.Error())
			}
		default:
			if tlv.IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			}
		}
	}

	return c, nil
}

// Encode encodes a FaceQueryFilter.
func (f *FaceQueryFilter) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.FaceQueryFilter)

	if f.FaceID != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.FaceID, *f.FaceID))
	}
	if f.URIScheme != nil {
		wire.Append(tlv.NewBlock(tlv.URIScheme, []byte(*f.URIScheme)))
	}
	if f.URI != nil {
		wire.Append(tlv.NewBlock(tlv.URI, []byte(f.URI.String())))
	}
	if f.LocalURI != nil {
		wire.Append(tlv.NewBlock(tlv.LocalURI, []byte(f.LocalURI.String())))
	}
	if f.FaceScope != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.FaceScope, *f.FaceScope))
	}
	if f.FacePersistency != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.FacePersistency, *f.FacePersistency))
	}
	if f.LinkType != nil {
		wire.Append(tlv.EncodeNNIBlock(tlv.LinkType, *f.LinkType))
	}

	wire.Encode()
	return wire, nil
}
