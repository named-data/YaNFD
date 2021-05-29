/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"errors"
	"strconv"
	"time"

	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/named-data/YaNFD/ndn/util"
)

// MetaInfo represents the MetaInfo in a Data packet.
type MetaInfo struct {
	contentType     *uint64
	freshnessPeriod *time.Duration
	finalBlockID    NameComponent
	wire            *tlv.Block
}

// NewMetaInfo creates a new MetaInfo structure.
func NewMetaInfo() *MetaInfo {
	m := new(MetaInfo)
	return m
}

// DecodeMetaInfo decodes a MetaInfo from a block.
func DecodeMetaInfo(wire *tlv.Block) (*MetaInfo, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}
	if len(wire.Subelements()) == 0 {
		wire.Parse()
	}

	m := new(MetaInfo)
	m.wire = wire
	mostRecentElem := 0
	var err error
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case tlv.ContentType:
			if mostRecentElem >= 1 {
				return nil, errors.New("ContentType is duplicate or out-of-order")
			}
			mostRecentElem = 1
			m.contentType = new(uint64)
			*m.contentType, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("error decoding ContentType")
			}
		case tlv.FreshnessPeriod:
			if mostRecentElem >= 2 {
				return nil, errors.New("FreshnessPeriod is duplicate or out-of-order")
			}
			mostRecentElem = 2
			freshnessPeriod, err := tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("error decoding FreshnessPeriod")
			}
			m.freshnessPeriod = new(time.Duration)
			*m.freshnessPeriod = time.Duration(freshnessPeriod) * time.Millisecond
		case tlv.FinalBlockID:
			if mostRecentElem == 3 {
				return nil, errors.New("FinalBlockId is duplicate or out-or-order")
			}
			mostRecentElem = 3
			if len(elem.Subelements()) == 0 {
				elem.Parse()
			}
			if len(elem.Subelements()) != 1 {
				return nil, errors.New("FinalBlockId must contain exactly one name component")
			}
			m.finalBlockID, err = DecodeNameComponent(elem.Subelements()[0])
			if err != nil {
				return nil, errors.New("error decoding FinalBlockId")
			}
		default:
			if tlv.IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			}
			// If non-critical, ignore
		}
	}
	return m, nil
}

func (m *MetaInfo) String() string {
	str := "MetaInfo("

	isFirst := true
	if m.contentType != nil {
		if !isFirst {
			str += ", "
		}
		str += "ContentType=" + strconv.FormatUint(*m.contentType, 10)
		isFirst = false
	}
	if m.freshnessPeriod != nil {
		if !isFirst {
			str += ", "
		}
		str += "FreshnessPeriod=" + strconv.FormatInt(m.freshnessPeriod.Milliseconds(), 10) + "ms"
	}
	if m.finalBlockID != nil {
		if !isFirst {
			str += ", "
		}
		str += "FinalBlockId=" + m.finalBlockID.String()
	}

	str += ")"
	return str
}

// ContentType returns the ContentType set in the MetaInfo.
func (m *MetaInfo) ContentType() *uint64 {
	return m.contentType
}

// SetContentType sets the ContentType in the MetaInfo.
func (m *MetaInfo) SetContentType(contentType uint64) {
	m.contentType = new(uint64)
	*m.contentType = contentType
	m.wire = nil
}

// UnsetContentType unsets the ContentType in the MetaInfo.
func (m *MetaInfo) UnsetContentType() {
	m.contentType = nil
	m.wire = nil
}

// FreshnessPeriod returns the FreshnessPeriod set in the MetaInfo.
func (m *MetaInfo) FreshnessPeriod() *time.Duration {
	return m.freshnessPeriod
}

// SetFreshnessPeriod sets the FreshnessPeriod in the MetaInfo.
func (m *MetaInfo) SetFreshnessPeriod(freshnessPeriod time.Duration) {
	m.freshnessPeriod = new(time.Duration)
	*m.freshnessPeriod = freshnessPeriod
	m.wire = nil
}

// UnsetFreshnessPeriod unsets the FreshnessPeriod in the MetaInfo.
func (m *MetaInfo) UnsetFreshnessPeriod() {
	m.freshnessPeriod = nil
	m.wire = nil
}

// FinalBlockID returns the FinalBlockId set in the MetaInfo.
func (m *MetaInfo) FinalBlockID() NameComponent {
	return m.finalBlockID
}

// SetFinalBlockID sets the FinalBlockId in the MetaInfo.
func (m *MetaInfo) SetFinalBlockID(finalBlockID NameComponent) {
	m.finalBlockID = finalBlockID
	m.wire = nil
}

// UnsetFinalBlockID unsets the FinalBlockId in the MetaInfo.
func (m *MetaInfo) UnsetFinalBlockID() {
	m.finalBlockID = nil
	m.wire = nil
}

// Encode encodes the MetaInfo into a block.
func (m *MetaInfo) Encode() (*tlv.Block, error) {
	if m.wire != nil {
		return m.wire, nil
	}

	m.wire = new(tlv.Block)
	m.wire.SetType(tlv.MetaInfo)

	// ContentType
	if m.contentType != nil {
		m.wire.Append(tlv.EncodeNNIBlock(tlv.ContentType, *m.contentType))
	}

	// FreshnessPeriod
	if m.freshnessPeriod != nil {
		m.wire.Append(tlv.EncodeNNIBlock(tlv.FreshnessPeriod, uint64(m.freshnessPeriod.Milliseconds())))
	}

	// FinalBlockId
	if m.finalBlockID != nil {
		encodedComponent, err := m.finalBlockID.Encode().Wire()
		if err != nil {
			return nil, errors.New("unable to encode FinalBlockId")
		}
		m.wire.Append(tlv.NewBlock(tlv.FinalBlockID, encodedComponent))
	}

	m.wire.Wire()
	return m.wire, nil
}

// HasWire returns whether the MetaInfo has an existing valid wire encoding.
func (m *MetaInfo) HasWire() bool {
	return m.wire != nil
}
