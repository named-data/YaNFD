/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"errors"

	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/ndn/util"
)

// Data represents an NDN Data packet.
type Data struct {
	name     Name
	metaInfo *MetaInfo
	content  []byte
	// TODO: Signature
	wire *tlv.Block
}

// NewData creates a new Data packet with the given name and content.
func NewData(name *Name, content []byte) *Data {
	if name == nil {
		return nil
	}

	d := new(Data)
	d.name = *name.DeepCopy()
	d.metaInfo = NewMetaInfo()
	d.content = make([]byte, len(content))
	copy(d.content, content)
	return d
}

// DecodeData decodes a Data packet from the wire.
func DecodeData(wire *tlv.Block) (*Data, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}
	wire.Parse()

	d := new(Data)
	d.wire = wire.DeepCopy()
	mostRecentElem := 0
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case tlv.Name:
			if mostRecentElem >= 1 {
				return nil, errors.New("Name is duplicate or out-of-order")
			}
			mostRecentElem = 1
			name, err := DecodeName(elem)
			if err != nil {
				return nil, errors.New("Error decoding Name")
			}
			d.name = *name
		case tlv.MetaInfo:
			if mostRecentElem >= 2 {
				return nil, errors.New("MetaInfo is duplicate or out-of-order")
			}
			mostRecentElem = 2
			metaInfo, err := DecodeMetaInfo(elem)
			if err != nil {
				return nil, err
			}
			d.metaInfo = metaInfo
		case tlv.Content:
			if mostRecentElem == 3 {
				return nil, errors.New("Content is duplicate or out-or-order")
			}
			mostRecentElem = 3
			d.content = make([]byte, len(elem.Value()))
			copy(d.content, elem.Value())
		// TODO: Signature
		default:
			if tlv.IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			}
			// If non-critical, ignore
		}
	}
	return d, nil
}

// DeepCopy returns a deep copy of the Data.
func (d *Data) DeepCopy() *Data {
	// TODO
	return nil
}

// Name returns the name of the Data packet.
func (d *Data) Name() *Name {
	return d.name.DeepCopy()
}

// SetName sets the name of the Data packet.
func (d *Data) SetName(name *Name) {
	d.name = *name.DeepCopy()
	d.wire = nil
}

// MetaInfo returns the MetaInfo of the Data packet.
func (d *Data) MetaInfo() *MetaInfo {
	return d.metaInfo.DeepCopy()
}

// SetMetaInfo sets the MetaInfo of the Data packet.
func (d *Data) SetMetaInfo(metaInfo *MetaInfo) error {
	if metaInfo == nil {
		return util.ErrOutOfRange
	}

	d.metaInfo = metaInfo.DeepCopy()
	d.wire = nil
	return nil
}

// Content returns a copy of the content in the Data packet.
func (d *Data) Content() []byte {
	content := make([]byte, len(d.content))
	copy(content, d.content)
	return content
}

// SetContent sets the content of the Data packet.
func (d *Data) SetContent(content []byte) {
	d.content = make([]byte, len(content))
	copy(d.content, content)
	d.wire = nil
}

// Encode encodes the Data into a block.
func (d *Data) Encode() (*tlv.Block, error) {
	if d.wire == nil {
		d.wire = tlv.NewEmptyBlock(tlv.Data)
		d.wire.Append(d.name.Encode())
		if d.metaInfo.ContentType() != nil || d.metaInfo.FreshnessPeriod() != nil || d.metaInfo.FinalBlockID() != nil {
			encodedMetaInfo, err := d.metaInfo.Encode()
			if err != nil {
				d.wire = nil
				return nil, errors.New("Unable to encode MetaInfo")
			}
			d.wire.Append(encodedMetaInfo)
		}
		d.wire.Append(tlv.NewBlock(tlv.Content, d.content))

		// TODO: Signature
	}

	d.wire.Wire()
	return d.wire.DeepCopy(), nil
}

// HasWire returns whether the Data packet has an existing valid wire encoding.
func (d *Data) HasWire() bool {
	return d.wire != nil
}
