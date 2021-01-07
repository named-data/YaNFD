/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/eric135/YaNFD/ndn/security"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/ndn/util"
)

// Data represents an NDN Data packet.
type Data struct {
	name                    *Name
	metaInfo                *MetaInfo
	content                 []byte
	sigInfo                 *SignatureInfo
	sigValue                []byte
	shouldValidateSignature bool
	wire                    *tlv.Block

	pitToken []byte
}

// NewData creates a new Data packet with the given name and content.
func NewData(name *Name, content []byte) *Data {
	if name == nil {
		return nil
	}

	d := new(Data)
	d.name = name
	d.metaInfo = NewMetaInfo()
	d.content = make([]byte, len(content))
	d.sigInfo = NewSignatureInfo(security.DigestSha256Type)
	copy(d.content, content)
	return d
}

// DecodeData decodes a Data packet from the wire.
func DecodeData(wire *tlv.Block, shouldValidateSignature bool) (*Data, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}

	d := new(Data)
	d.shouldValidateSignature = shouldValidateSignature
	d.wire = wire
	d.wire.Parse()
	mostRecentElem := 0
	var err error
	for _, elem := range d.wire.Subelements() {
		switch elem.Type() {
		case tlv.Name:
			if mostRecentElem >= 1 {
				return nil, errors.New("Name is duplicate or out-of-order")
			}
			mostRecentElem = 1
			d.name, err = DecodeName(elem)
			if err != nil {
				return nil, errors.New("Error decoding Name")
			}
		case tlv.MetaInfo:
			if mostRecentElem >= 2 {
				return nil, errors.New("MetaInfo is duplicate or out-of-order")
			}
			mostRecentElem = 2
			d.metaInfo, err = DecodeMetaInfo(elem)
			if err != nil {
				return nil, err
			}
		case tlv.Content:
			if mostRecentElem >= 3 {
				return nil, errors.New("Content is duplicate or out-or-order")
			}
			mostRecentElem = 3
			d.content = make([]byte, len(elem.Value()))
			copy(d.content, elem.Value())
		case tlv.SignatureInfo:
			if mostRecentElem >= 4 {
				return nil, errors.New("SignatureInfo is duplicate or out-of-order")
			}
			mostRecentElem = 4
			d.sigInfo, err = DecodeSignatureInfo(elem)
			if err != nil {
				return nil, errors.New("Error decoding SignatureInfo")
			}
		case tlv.SignatureValue:
			if mostRecentElem >= 5 {
				return nil, errors.New("SignatureValue is duplicate or out-of-order")
			}
			mostRecentElem = 5
			d.sigValue = make([]byte, len(elem.Value()))
			copy(d.sigValue, elem.Value())
		default:
			if tlv.IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			}
			// If non-critical, ignore
		}
	}

	if d.name == nil || d.sigInfo == nil || len(d.sigValue) == 0 {
		fmt.Println(d.name, d.sigInfo, d.sigValue)
		return nil, errors.New("Data missing required field")
	}

	if d.shouldValidateSignature {
		isSignatureValid, err := d.validateSignature()
		if err != nil {
			return nil, err
		}
		if !isSignatureValid {
			return nil, errors.New("Unable to validate signature in decoded Data")
		}
	}

	return d, nil
}

func (d *Data) String() string {
	str := "Data(" + d.name.String()
	if d.metaInfo != nil {
		str += ", " + d.metaInfo.String()
	}
	str += ", ContentLen=" + strconv.FormatInt(int64(len(d.content)), 10) + ")"
	return str
}

// Name returns the name of the Data packet.
func (d *Data) Name() *Name {
	return d.name
}

// SetName sets the name of the Data packet.
func (d *Data) SetName(name *Name) {
	d.name = name
	d.wire = nil
	d.sigValue = make([]byte, 0)
}

// MetaInfo returns the MetaInfo of the Data packet.
func (d *Data) MetaInfo() *MetaInfo {
	return d.metaInfo
}

// SetMetaInfo sets the MetaInfo of the Data packet.
func (d *Data) SetMetaInfo(metaInfo *MetaInfo) {
	d.metaInfo = metaInfo
	d.wire = nil
	d.sigValue = make([]byte, 0)
}

// Content returns a copy of the content in the Data packet.
func (d *Data) Content() []byte {
	return d.content
}

// SetContent sets the content of the Data packet.
func (d *Data) SetContent(content []byte) {
	d.content = content
	d.wire = nil
	d.sigValue = make([]byte, 0)
}

// SignatureInfo returns a copy of the SignatureInfo in the Data packet.
func (d *Data) SignatureInfo() *SignatureInfo {
	return d.sigInfo
}

// SetSignatureInfo sets the SignatureInfo in the Data packet.
func (d *Data) SetSignatureInfo(sigInfo *SignatureInfo) {
	d.sigInfo = sigInfo
	d.wire = nil
	d.sigValue = make([]byte, 0)
}

// SignatureValue returns a copy of the SignatureValue in the Data packet. If the signature has not yet been calculated,
func (d *Data) SignatureValue() []byte {
	if len(d.sigValue) == 0 && d.computeSignatureValue() != nil {
		return make([]byte, 0)
	}

	return d.sigValue
}

// ShouldValidateSignature returns whether signature validation is enabled for this Data.
func (d *Data) ShouldValidateSignature() bool {
	return d.shouldValidateSignature
}

func (d *Data) validateSignature() (bool, error) {
	if d.wire == nil {
		_, err := d.Encode()
		if err != nil {
			// Can't validate signature if can't encode packet
			return false, errors.New("Cannot encode packet")
		}
	}

	if d.wire.Find(tlv.SignatureInfo) == nil {
		// No SignatureInfo!
		return false, errors.New("SignatureInfo not present")
	}

	d.wire.Parse()
	wire, err := d.wire.Wire()
	if err != nil {
		return false, err
	}
	buffer := make([]byte, 0, len(wire))
	for _, elem := range d.wire.Subelements() {
		if elem.Type() == tlv.SignatureValue {
			break
		}
		elemWire, err := elem.Wire()
		if err != nil {
			return false, err
		}
		buffer = append(buffer, elemWire...)
	}

	return security.Verify(d.SignatureInfo().Type(), buffer, d.SignatureValue())
}

func (d *Data) computeSignatureValue() error {
	if d.wire == nil {
		if _, err := d.Encode(); err != nil {
			return err
		}
	}
	if d.wire.Find(tlv.SignatureInfo) == nil {
		// No SignatureInfo!
		return errors.New("SignatureInfo missing")
	}

	wire, err := d.wire.Wire()
	if err != nil {
		return err
	}
	buffer := make([]byte, 0, len(wire))
	for _, elem := range d.wire.Subelements() {
		if elem.Type() == tlv.SignatureValue {
			break
		}
		elemWire, err := elem.Wire()
		if err != nil {
			return err
		}
		buffer = append(buffer, elemWire...)
	}

	signature, err := security.Sign(d.SignatureInfo().Type(), buffer)
	if err == nil {
		d.sigValue = make([]byte, len(signature))
		copy(d.sigValue, signature)
	}
	return err
}

// Encode encodes the Data into a block.
func (d *Data) Encode() (*tlv.Block, error) {
	if d.wire == nil {
		d.wire = tlv.NewEmptyBlock(tlv.Data)
		d.wire.Append(d.name.Encode())
		if d.metaInfo.contentType != nil || d.metaInfo.freshnessPeriod != nil || d.metaInfo.finalBlockID != nil {
			encodedMetaInfo, err := d.metaInfo.Encode()
			if err != nil {
				d.wire = nil
				return nil, errors.New("Unable to encode MetaInfo")
			}
			d.wire.Append(encodedMetaInfo)
		}
		d.wire.Append(tlv.NewBlock(tlv.Content, d.content))

		sigInfo, err := d.sigInfo.Encode()
		if err != nil {
			d.wire = nil
			return nil, errors.New("Unable to encode SignatureInfo")
		}
		d.wire.Append(sigInfo)

		if d.computeSignatureValue() != nil {
			d.wire = nil
			return nil, errors.New("Unable to encode SignatureValue")
		}
		d.wire.Append(tlv.NewBlock(tlv.SignatureValue, d.sigValue))
	}

	d.wire.Wire()
	return d.wire, nil
}

// HasWire returns whether the Data packet has an existing valid wire encoding.
func (d *Data) HasWire() bool {
	return d.wire != nil
}

// PitToken returns the PIT token attached to the Data (if any).
func (d *Data) PitToken() []byte {
	return d.pitToken
}

// SetPitToken sets the PIT token attached to the Data.
func (d *Data) SetPitToken(pitToken []byte) {
	d.pitToken = pitToken
}
