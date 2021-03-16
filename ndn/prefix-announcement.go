/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"bytes"
	"errors"
	"time"

	"github.com/eric135/YaNFD/ndn/tlv"
)

// PrefixAnnouncement is a specially-formatted Data packet used by applications to announce prefixes they produce.
type PrefixAnnouncement struct {
	data *Data // Underlying data
}

// NewPrefixAnnouncement creates a prefix announcement from a data packet.
func NewPrefixAnnouncement(data *Data) (*PrefixAnnouncement, error) {
	p := new(PrefixAnnouncement)
	p.data = data
	if !p.Valid() {
		return nil, errors.New("PrefixAnnouncement name does not meet requirements")
	}
	return p, nil
}

// DecodePrefixAnnouncement decodes a prefix announcement from the wire.
func DecodePrefixAnnouncement(wire *tlv.Block) (*PrefixAnnouncement, error) {
	p := new(PrefixAnnouncement)
	var err error
	p.data, err = DecodeData(wire, false)
	if err != nil {
		return nil, err
	}

	// Validate Data name
	if !p.Valid() {
		return nil, errors.New("PrefixAnnouncement name does not meet requirements")
	}

	return p, nil
}

// Valid returns whether the PrefixAnnouncement is validly formatted.
func (p *PrefixAnnouncement) Valid() bool {
	if p.data.Name().Size() < 4 {
		return false
	}

	keyword, ok := p.data.Name().At(-3).(*KeywordNameComponent)
	if !ok || !bytes.Equal(keyword.value, []byte("PA")) {
		return false
	}

	_, ok = p.data.Name().At(-2).(*VersionNameComponent)
	if !ok {
		return false
	}

	segment, ok := p.data.Name().At(-1).(*SegmentNameComponent)
	if !ok || segment.rawValue != 0 {
		return false
	}

	if p.data.MetaInfo() == nil || p.data.MetaInfo().ContentType() == nil || *p.data.MetaInfo().ContentType() != 5 {
		return false
	}

	if p.data.SignatureInfo() == nil || p.data.SignatureValue() == nil {
		return false
	}

	// Validate content
	content := tlv.NewBlock(tlv.Content, p.data.Content())
	if content.Parse() != nil {
		return false
	}

	expirationPeriod := content.Find(tlv.ExpirationPeriod)
	if expirationPeriod == nil {
		return false
	}

	return true
}

// Prefix returns the prefix announced by the the prefix announcement.
func (p *PrefixAnnouncement) Prefix() *Name {
	return p.data.Name().Prefix(p.data.Name().Size() - 3)
}

// ExpirationPeriod returns the expiration period contained in the prefix announcement.
func (p *PrefixAnnouncement) ExpirationPeriod() uint64 {
	if !p.Valid() {
		return 0
	}

	content := tlv.NewBlock(tlv.Content, p.data.Content())
	if content.Parse() != nil {
		return 0
	}

	value, err := tlv.DecodeNNIBlock(content.Find(tlv.ExpirationPeriod))
	if err != nil {
		return 0
	}

	return value
}

// ValidityPeriod returns the validility period contained in the prefix announcement. If unset, returns 0 time for both values.
func (p *PrefixAnnouncement) ValidityPeriod() (time.Time, time.Time) {
	if !p.Valid() {
		return time.Unix(0, 0), time.Unix(0, 0)
	}

	content := tlv.NewBlock(tlv.Content, p.data.Content())
	if content.Parse() != nil {
		return time.Unix(0, 0), time.Unix(0, 0)
	}

	validityPeriod := content.Find(tlv.ValidityPeriod)
	if validityPeriod != nil {
		return time.Unix(0, 0), time.Unix(0, 0)
	}
	validityPeriod.Parse()

	notBeforeBlock := validityPeriod.Find(tlv.NotBefore)
	notAfterBlock := validityPeriod.Find(tlv.NotAfter)
	if notBeforeBlock == nil || notAfterBlock == nil {
		return time.Unix(0, 0), time.Unix(0, 0)
	}

	layoutString := "20060102T150405"

	notBefore, err := time.Parse(layoutString, string(notBeforeBlock.Value()))
	if err != nil {
		return time.Unix(0, 0), time.Unix(0, 0)
	}

	notAfter, err := time.Parse(layoutString, string(notAfterBlock.Value()))
	if err != nil {
		return time.Unix(0, 0), time.Unix(0, 0)
	}

	return notBefore, notAfter
}
