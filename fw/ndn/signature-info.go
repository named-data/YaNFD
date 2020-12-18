/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/eric135/YaNFD/ndn/security"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/ndn/util"
)

// SignatureInfo represents either the SignatureInfo (for Data packets) or InterestSignatureInfo blocks.
type SignatureInfo struct {
	signatureType security.SignatureType
	keyLocator    *tlv.Block
	nonce         []byte
	time          *time.Time
	seqNum        *uint64
	isInterest    bool
	wire          *tlv.Block
}

// NewSignatureInfo creates a new SignatureInfo for a Data packet.
func NewSignatureInfo(signatureType security.SignatureType) *SignatureInfo {
	s := new(SignatureInfo)
	s.signatureType = signatureType
	s.isInterest = false
	return s
}

// NewInterestSignatureInfo creates a new InterestSignatureInfo.
func NewInterestSignatureInfo(signatureType security.SignatureType) *SignatureInfo {
	s := new(SignatureInfo)
	s.signatureType = signatureType
	s.isInterest = true
	return s
}

// DecodeSignatureInfo decodes a SignatureInfo or InterestSignatureInfo from the wire.
func DecodeSignatureInfo(wire *tlv.Block) (*SignatureInfo, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}

	if wire.Type() != tlv.SignatureInfo && wire.Type() != tlv.InterestSignatureInfo {
		return nil, errors.New("Block must be SignatureInfo or InterestSignatureInfo")
	}

	s := new(SignatureInfo)
	// We already ensured is SignatureInfo or InterestSignatureInfo
	s.isInterest = wire.Type() == tlv.InterestSignatureInfo
	s.wire = wire.DeepCopy()
	s.wire.Parse()
	mostRecentElem := 0
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case tlv.SignatureType:
			if mostRecentElem >= 1 {
				return nil, errors.New("SignatureType is duplicate or out-of-order")
			}
			mostRecentElem = 1
			signatureType, err := tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Error decoding SignatureType")
			}
			s.signatureType = security.SignatureType(signatureType)
		case tlv.KeyLocator:
			if mostRecentElem >= 2 {
				return nil, errors.New("KeyLocator is duplicate or out-of-order")
			}
			mostRecentElem = 2
			s.keyLocator = elem.DeepCopy()
		case tlv.SignatureNonce:
			if mostRecentElem >= 3 {
				return nil, errors.New("SignatureNonce is duplicate or out-or-order")
			}
			mostRecentElem = 3

			if !s.isInterest {
				return nil, errors.New("SignatureNonce cannot be present in SignatureInfo for Data")
			}
			s.nonce = make([]byte, len(elem.Value()))
			copy(s.nonce, elem.Value())
		case tlv.SignatureTime:
			if mostRecentElem >= 4 {
				return nil, errors.New("SignatureTime is duplicate or out-or-order")
			}
			mostRecentElem = 4

			if !s.isInterest {
				return nil, errors.New("SignatureTime cannot be present in SignatureInfo for Data")
			}
			timeMS, err := tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Error decoding SignatureTime")
			}
			s.time = new(time.Time)
			*s.time = time.Unix(int64(timeMS/1000), int64(timeMS*1000000))
		case tlv.SignatureSeqNum:
			if mostRecentElem >= 5 {
				return nil, errors.New("SignatureSeqNum is duplicate or out-or-order")
			}
			mostRecentElem = 5

			if !s.isInterest {
				return nil, errors.New("SignatureSeqNum cannot be present in SignatureInfo for Data")
			}
			seqNum, err := tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Error decoding SignatureSeqNum")
			}
			s.seqNum = new(uint64)
			*s.seqNum = seqNum
		default:
			if tlv.IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			}
			// If non-critical, ignore
		}
	}

	return s, nil
}

func (s *SignatureInfo) String() string {
	str := ""
	if s.isInterest {
		str += "InterestSignatureInfo("
	} else {
		str += "SignatureInfo("
	}

	str += "SignatureType=" + strconv.FormatUint(uint64(s.signatureType), 10)

	if s.keyLocator != nil {
		str += ", KeyLocator"
	}
	if len(s.nonce) > 0 {
		str += ", SignatureNonce=0x" + hex.EncodeToString(s.nonce)
	}
	if s.time != nil {
		str += ", SignatureTime=" + s.time.String()
	}
	if s.seqNum != nil {
		str += ", SignatureSeqNum=" + strconv.FormatUint(*s.seqNum, 10)
	}

	str += ")"
	return str
}

// DeepCopy creates a deep copy of the SignatureInfo.
func (s *SignatureInfo) DeepCopy() *SignatureInfo {
	copyS := new(SignatureInfo)

	copyS.signatureType = s.signatureType
	if s.keyLocator != nil {
		copyS.keyLocator = s.keyLocator.DeepCopy()
	}
	copyS.nonce = make([]byte, len(s.nonce))
	copy(copyS.nonce, s.nonce)
	if s.time != nil {
		copyS.time = new(time.Time)
		*copyS.time = time.Unix(s.time.Unix(), s.time.UnixNano())
	}
	if s.seqNum != nil {
		copyS.seqNum = new(uint64)
		*copyS.seqNum = *s.seqNum
	}

	return copyS
}

// Type returns the type of the signature.
func (s *SignatureInfo) Type() security.SignatureType {
	return s.signatureType
}

// SetType sets the type of the signature.
func (s *SignatureInfo) SetType(signatureType security.SignatureType) {
	s.signatureType = signatureType
	s.wire = nil
}

// KeyLocator returns the KeyLocator of the signature.
func (s *SignatureInfo) KeyLocator() *tlv.Block {
	if s.keyLocator == nil {
		return nil
	}
	return s.keyLocator.DeepCopy()
}

// SetKeyLocator sets the KeyLocator of the signature.
func (s *SignatureInfo) SetKeyLocator(keyLocator *tlv.Block) {
	s.wire = nil
	if keyLocator == nil {
		s.keyLocator = nil
		return
	}
	s.keyLocator = keyLocator.DeepCopy()
}

// UnsetKeyLocator unsets the KeyLocator of the signature.
func (s *SignatureInfo) UnsetKeyLocator() {
	s.keyLocator = nil
	s.wire = nil
}

// Nonce returns the SignatureNonce of the signature.
func (s *SignatureInfo) Nonce() []byte {
	nonce := make([]byte, len(s.nonce))
	copy(nonce, s.nonce)
	return nonce
}

// SetNonce sets the SignatureNonce of the signature.
func (s *SignatureInfo) SetNonce(nonce []byte) {
	s.nonce = make([]byte, len(nonce))
	copy(s.nonce, nonce)
	s.wire = nil
}

// UnsetNonce unsets the SignatureNonce of the signature.
func (s *SignatureInfo) UnsetNonce() {
	s.nonce = make([]byte, 0)
	s.wire = nil
}

// Time returns the SignatureTime of the signature.
func (s *SignatureInfo) Time() *time.Time {
	if s.keyLocator == nil {
		return nil
	}
	newTime := time.Unix(s.time.Unix(), s.time.UnixNano())
	return &newTime
}

// SetTime sets the SignatureTime of the signature.
func (s *SignatureInfo) SetTime(newTime *time.Time) {
	s.wire = nil
	if newTime == nil {
		s.time = nil
		return
	}
	s.time = new(time.Time)
	*s.time = time.Unix(s.time.Unix(), s.time.UnixNano())
}

// UnsetTime unsets the SignatureTime of the signature.
func (s *SignatureInfo) UnsetTime() {
	s.time = nil
	s.wire = nil
}

// SeqNum returns the SignatureSeqNum of the signature.
func (s *SignatureInfo) SeqNum() *uint64 {
	if s.seqNum == nil {
		return nil
	}
	seqNum := new(uint64)
	*seqNum = *s.seqNum
	return seqNum
}

// SetSeqNum sets the SignatureSeqNum of the signature.
func (s *SignatureInfo) SetSeqNum(seqNum uint64) {
	s.seqNum = new(uint64)
	*s.seqNum = seqNum
	s.wire = nil
}

// UnsetSeqNum unsets the SignatureSeqNum of the signature.
func (s *SignatureInfo) UnsetSeqNum() {
	s.seqNum = nil
	s.wire = nil
}

// Interest returns true if this is an InterestSignatureInfo and false if it is a SignatureInfo (for Data packets).
func (s *SignatureInfo) Interest() bool {
	return s.isInterest
}

// Encode encodes the SignatureInfo (or InterestSignatureInfo) into a block.
func (s *SignatureInfo) Encode() (*tlv.Block, error) {
	if s.wire == nil {
		if s.isInterest {
			s.wire = tlv.NewEmptyBlock(tlv.InterestSignatureInfo)
		} else {
			s.wire = tlv.NewEmptyBlock(tlv.SignatureInfo)
		}
		s.wire.Append(tlv.EncodeNNIBlock(tlv.SignatureType, uint64(s.signatureType)))

		if s.keyLocator != nil {
			keyLocatorWire, err := s.keyLocator.Wire()
			if err != nil {
				return nil, errors.New("Unable to encode KeyLocator")
			}
			s.wire.Append(tlv.NewBlock(tlv.KeyLocator, keyLocatorWire))
		}

		if s.isInterest {
			if s.nonce != nil {
				s.wire.Append(tlv.NewBlock(tlv.SignatureNonce, s.nonce))
			}
			if s.time != nil {
				s.wire.Append(tlv.EncodeNNIBlock(tlv.SignatureTime, uint64(s.time.UnixNano()/1000000)))
			}
			if s.seqNum != nil {
				s.wire.Append(tlv.EncodeNNIBlock(tlv.SignatureSeqNum, *s.seqNum))
			}
		}
	}

	s.wire.Wire()
	return s.wire.DeepCopy(), nil
}

// HasWire returns whether a valid up-to-date wire encoding exists for the SignatureInfo.
func (s *SignatureInfo) HasWire() bool {
	return s.wire != nil
}
