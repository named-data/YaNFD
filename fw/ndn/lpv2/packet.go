/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package lpv2

import (
	"errors"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// Packet represents an NDNLPv2 frame.
type Packet struct {
	sequence           *uint64
	fragIndex          *uint64
	fragCount          *uint64
	pitToken           []byte
	nextHopFaceID      *uint64
	incomingFaceID     *uint64
	cachePolicyType    *uint64
	congestionMark     *uint64
	txSequence         *uint64
	acks               []uint64
	nonDiscovery       bool
	prefixAnnouncement *ndn.Data
	fragment           *tlv.Block
	wire               *tlv.Block
}

// NewPacket returns an NDNLPv2 frame containing a copy of the provided network-layer packet.
func NewPacket(fragment []byte) *Packet {
	p := new(Packet)
	p.fragment = tlv.NewBlock(Fragment, fragment)
	return p
}

// NewIDLEPacket returns an NDNLPv2 IDLE frame.
func NewIDLEPacket() *Packet {
	p := new(Packet)
	return p
}

// DecodePacket returns an NDNLPv2 frame decoded from the wire.
func DecodePacket(wire *tlv.Block) (*Packet, error) {
	if wire == nil {
		return nil, errors.New("Wire is unset")
	}

	p := new(Packet)

	// If type is not LpPacket, then this is a "bare" packet.
	if wire.Type() != LpPacket {
		p.fragment = wire
		return p, nil
	}

	p.wire = wire.DeepCopy()
	p.wire.Parse()
	var err error
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case Fragment:
			p.fragment = elem
		case Sequence:
			*p.sequence, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode Sequence")
			}
		case FragIndex:
			*p.fragIndex, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode FragIndex")
			}
		case FragCount:
			*p.fragCount, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode FragCount")
			}
		case PitToken:
			p.pitToken = make([]byte, len(elem.Value()))
			copy(p.pitToken, elem.Value())
		case NextHopFaceID:
			*p.nextHopFaceID, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode NextHopFaceId")
			}
		case IncomingFaceID:
			*p.incomingFaceID, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode IncomingFaceId")
			}
		case CachePolicy:
			elem.Parse()
			if cachePolicyTypeBlock := elem.Find(CachePolicyType); cachePolicyTypeBlock != nil {
				*p.cachePolicyType, err = tlv.DecodeNNIBlock(cachePolicyTypeBlock)
				if err != nil {
					return nil, errors.New("Unable to decode CachePolicyType")
				}
			} else {
				return nil, errors.New("CachePolicy element does not contain CachePolicyType")
			}
		case CongestionMark:
			*p.congestionMark, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode CongestionMark")
			}
		case Ack:
			ack, err := tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode Ack")
			}
			p.acks = append(p.acks, ack)
		case TxSequence:
			*p.txSequence, err = tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Unable to decode TxSequence")
			}
		case NonDiscovery:
			p.nonDiscovery = true
		case PrefixAnnouncement:
			if err := elem.Parse(); err != nil {
				return nil, errors.New("Unable to parse PrefixAnnouncement: " + err.Error())
			}
			if elem.Find(tlv.Data) == nil {
				return nil, errors.New("PrefixAnnouncement does not contain Data")
			}
			p.prefixAnnouncement, err = ndn.DecodeData(elem.Find(tlv.Data), false)
			if err != nil {
				return nil, errors.New("Unable to decode PrefixAnnouncement: " + err.Error())
			}
		case Nack:
			// Gracefully ignore Nacks
		default:
			if IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			}
		}
	}

	return p, nil
}

// IsBare returns whether the LpPacket only contains a fragment and has no headers fields.
func (p *Packet) IsBare() bool {
	return p.sequence == nil && p.fragIndex == nil && p.fragCount == nil && len(p.pitToken) == 0 && p.nextHopFaceID == nil && p.incomingFaceID == nil && p.cachePolicyType == nil && p.congestionMark == nil && p.txSequence == nil && len(p.acks) == 0 && p.nonDiscovery == false && p.prefixAnnouncement == nil && p.prefixAnnouncement != nil
}

// IsIdle returns whether the LpPacket is an "IDLE" frame and does not contain a fragment.
func (p *Packet) IsIdle() bool {
	return p.fragment == nil
}

// Sequence returns the Sequence of the LpPacket or nil if it is unset.
func (p *Packet) Sequence() *uint64 {
	if p.sequence == nil {
		return nil
	}

	sequence := new(uint64)
	*sequence = *p.sequence
	return sequence
}

// SetSequence sets the Sequence of the LpPacket.
func (p *Packet) SetSequence(sequence uint64) {
	*p.sequence = sequence
	p.wire = nil
}

// FragIndex returns the FragIndex of the LpPacket or nil if it is unset.
func (p *Packet) FragIndex() *uint64 {
	if p.fragIndex == nil {
		return nil
	}

	fragIndex := new(uint64)
	*fragIndex = *p.fragIndex
	return fragIndex
}

// SetFragIndex sets the FragIndex of the LpPacket.
func (p *Packet) SetFragIndex(fragIndex uint64) {
	*p.fragIndex = fragIndex
	p.wire = nil
}

// FragCount returns the FragCount of the LpPacket or nil if it is unset.
func (p *Packet) FragCount() *uint64 {
	if p.fragCount == nil {
		return nil
	}

	fragCount := new(uint64)
	*fragCount = *p.fragCount
	return fragCount
}

// SetFragCount sets the FragCount of the LpPacket.
func (p *Packet) SetFragCount(fragCount uint64) {
	*p.fragCount = fragCount
	p.wire = nil
}

// PitToken returns the PitToken set in the LpPacket or an empty slice if it is unset.
func (p *Packet) PitToken() []byte {
	pitToken := make([]byte, len(p.pitToken))
	copy(pitToken, p.pitToken)
	return pitToken
}

// SetPitToken sets the PitToken of the LpPacket.
func (p *Packet) SetPitToken(pitToken []byte) {
	p.pitToken = make([]byte, len(pitToken))
	copy(p.pitToken, pitToken)
	p.wire = nil
}

// NextHopFaceID returns the NextHopFaceId of the LpPacket or nil if it is unset.
func (p *Packet) NextHopFaceID() *uint64 {
	if p.nextHopFaceID == nil {
		return nil
	}

	nextHopFaceID := new(uint64)
	*nextHopFaceID = *p.nextHopFaceID
	return nextHopFaceID
}

// SetNextHopFaceID sets the NextHopFaceId of the LpPacket.
func (p *Packet) SetNextHopFaceID(nextHopFaceID uint64) {
	*p.nextHopFaceID = nextHopFaceID
	p.wire = nil
}

// IncomingFaceID returns the IncomingFaceId of the LpPacket or nil if it is unset.
func (p *Packet) IncomingFaceID() *uint64 {
	if p.incomingFaceID == nil {
		return nil
	}

	incomingFaceID := new(uint64)
	*incomingFaceID = *p.incomingFaceID
	return incomingFaceID
}

// SetIncomingFaceID sets the IncomingFaceId of the LpPacket.
func (p *Packet) SetIncomingFaceID(incomingFaceID uint64) {
	*p.incomingFaceID = incomingFaceID
	p.wire = nil
}

// CachePolicyType returns the CachePolicyType of the LpPacket or nil if it is unset.
func (p *Packet) CachePolicyType() *uint64 {
	if p.cachePolicyType == nil {
		return nil
	}

	cachePolicyType := new(uint64)
	*cachePolicyType = *p.cachePolicyType
	return cachePolicyType
}

// SetCachePolicytype sets the CachePolicyType of the LpPacket.
func (p *Packet) SetCachePolicytype(cachePolicyType uint64) {
	*p.cachePolicyType = cachePolicyType
	p.wire = nil
}

// CongestionMark returns the CongestionMark of the LpPacket or nil if it is unset.
func (p *Packet) CongestionMark() *uint64 {
	if p.congestionMark == nil {
		return nil
	}

	congestionMark := new(uint64)
	*congestionMark = *p.congestionMark
	return congestionMark
}

// SetCongestionMark sets the CongestionMark of the LpPacket.
func (p *Packet) SetCongestionMark(congestionMark uint64) {
	*p.congestionMark = congestionMark
	p.wire = nil
}

// TxSequence returns the TxSequence of the LpPacket or nil if it is unset.
func (p *Packet) TxSequence() *uint64 {
	if p.txSequence == nil {
		return nil
	}

	txSequence := new(uint64)
	*txSequence = *p.txSequence
	return txSequence
}

// SetTxSequence sets the TxSequence of the LpPacket.
func (p *Packet) SetTxSequence(txSequence uint64) {
	*p.txSequence = txSequence
	p.wire = nil
}

// Acks returns the Ack field(s) set in the LpPacket (if any).
func (p *Packet) Acks() []uint64 {
	acks := make([]uint64, len(p.acks))
	copy(acks, p.acks)
	return acks
}

// AppendAck appends an Ack to the LpPacket.
func (p *Packet) AppendAck(ack uint64) {
	p.acks = append(p.acks, ack)
	p.wire = nil
}

// ClearAcks removes all Acks from the LpPacket.
func (p *Packet) ClearAcks() {
	p.acks = make([]uint64, 0)
	p.wire = nil
}

// NonDiscovery returns whether the NonDiscovery flag is set in the LpPacket
func (p *Packet) NonDiscovery() bool {
	return p.nonDiscovery
}

// SetNonDiscovery sets the NonDiscovery flag of the LpPacket.
func (p *Packet) SetNonDiscovery(nonDiscovery bool) {
	p.nonDiscovery = nonDiscovery
	p.wire = nil
}

// PrefixAnnouncement returns the PrefixAnnouncement field of the LpPacket or nil if none is present.
func (p *Packet) PrefixAnnouncement() *ndn.Data {
	if p.prefixAnnouncement == nil {
		return nil
	}
	return p.prefixAnnouncement
}

// SetPrefixAnnouncement sets the PrefixAnnouncement field of the LpPacket.
func (p *Packet) SetPrefixAnnouncement(prefixAnnouncement *ndn.Data) {
	if prefixAnnouncement == nil {
		p.prefixAnnouncement = nil
	} else {
		p.prefixAnnouncement = prefixAnnouncement
	}
	p.wire = nil
}

// Fragment returns the Fragment field of the LpPacket or nil if it is unset.
func (p *Packet) Fragment() *tlv.Block {
	if p.fragment == nil {
		return nil
	}
	return p.fragment
}

// SetFragment sets the Fragment field of the LpPacket.
func (p *Packet) SetFragment(fragment *tlv.Block) {
	if fragment == nil {
		p.fragment = nil
	} else {
		p.fragment = fragment
	}
	p.wire = nil
}

// Encode encodes the LpPacket into a block.
func (p *Packet) Encode() (*tlv.Block, error) {
	if p.wire == nil {
		p.wire = tlv.NewEmptyBlock(LpPacket)

		// Sequence
		if p.sequence != nil {
			p.wire.Append(tlv.EncodeNNIBlock(Sequence, *p.sequence))
		}

		// FragIndex
		if p.fragIndex != nil {
			p.wire.Append(tlv.EncodeNNIBlock(FragIndex, *p.fragIndex))
		}

		// FragCount
		if p.fragCount != nil {
			p.wire.Append(tlv.EncodeNNIBlock(FragCount, *p.fragCount))
		}

		// PitToken
		if len(p.pitToken) > 0 {
			p.wire.Append(tlv.NewBlock(PitToken, p.pitToken))
		}

		// NextHopFaceID
		if p.nextHopFaceID != nil {
			p.wire.Append(tlv.EncodeNNIBlock(NextHopFaceID, *p.nextHopFaceID))
		}

		// IncomingFaceID
		if p.incomingFaceID != nil {
			p.wire.Append(tlv.EncodeNNIBlock(IncomingFaceID, *p.incomingFaceID))
		}

		// CachePolicyType
		if p.cachePolicyType != nil {
			cachePolicyTypeBlock := tlv.EncodeNNIBlock(CachePolicyType, *p.cachePolicyType)
			cachePolicyTypeBlockWire, err := cachePolicyTypeBlock.Wire()
			if err != nil {
				return nil, errors.New("Unable to encode CachePolicyType")
			}
			p.wire.Append(tlv.NewBlock(CachePolicy, cachePolicyTypeBlockWire))
		}

		// CongestionMark
		if p.congestionMark != nil {
			p.wire.Append(tlv.EncodeNNIBlock(CongestionMark, *p.congestionMark))
		}

		// TxSequence
		if p.txSequence != nil {
			p.wire.Append(tlv.EncodeNNIBlock(TxSequence, *p.txSequence))
		}

		// Acks
		for _, ack := range p.acks {
			p.wire.Append(tlv.EncodeNNIBlock(Ack, ack))
		}

		// NonDiscovery
		if p.nonDiscovery {
			p.wire.Append(tlv.NewEmptyBlock(NonDiscovery))
		}

		// PrefixAnnouncement
		if p.prefixAnnouncement != nil {
			prefixAnnouncementBlock, err := p.prefixAnnouncement.Encode()
			if err != nil {
				return nil, errors.New("Unable to encode PrefixAnnouncement: " + err.Error())
			}
			prefixAnnouncementWire, err := prefixAnnouncementBlock.Wire()
			if err != nil {
				return nil, errors.New("Unable to encode PrefixAnnouncement: " + err.Error())
			}
			p.wire.Append(tlv.NewBlock(PrefixAnnouncement, prefixAnnouncementWire))
		}

		// Fragment
		if p.fragment != nil {
			fragmentWire, err := p.fragment.Wire()
			if err != nil {
				return nil, errors.New("Unable to encode Fragment: " + err.Error())
			}
			p.wire.Append(tlv.NewBlock(Fragment, fragmentWire))
		}
	}
	p.wire.Wire()
	return p.wire, nil
}
