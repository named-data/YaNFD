/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package lpv2

import (
	"errors"
	"fmt"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/tlv"
)

func cloneUint64Ptr(input *uint64) *uint64 {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

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
	fragment           []byte
	wire               *tlv.Block
}

// NewPacket returns an NDNLPv2 frame containing a copy of the provided network-layer packet.
func NewPacket(fragment []byte) *Packet {
	fragmentCopy := make([]byte, len(fragment))
	copy(fragmentCopy, fragment)
	return &Packet{
		fragment: fragmentCopy,
	}
}
func NewPacketNoCopy(fragment []byte) *Packet {
	return &Packet{
		fragment: fragment,
	}
}

// NewIDLEPacket returns an NDNLPv2 IDLE frame.
func NewIDLEPacket() *Packet {
	return &Packet{}
}

func DecodePacketNoCopy(wire *tlv.Block) (*Packet, error) {
	if wire == nil {
		return nil, errors.New("wire is unset")
	}

	p := &Packet{}

	// If type is not LpPacket, then this is a "bare" packet.
	if wire.Type() != LpPacket {
		encodedFragment, err := wire.Wire()
		if err != nil {
			return nil, errors.New("unable to encode bare fragment: " + err.Error())
		}
		// copy encoded fragment
		p.fragment = make([]byte, len(encodedFragment))
		copy(p.fragment, encodedFragment)
		return p, nil
	}

	p.wire = wire
	if e := p.wire.Parse(); e != nil {
		return nil, errors.New("unable to decode LpPacket")
	}
	for _, elem := range p.wire.Subelements() {
		switch elem.Type() {
		case Fragment:
			val := elem.Value()
			p.fragment = make([]byte, len(val))
			copy(p.fragment, val)
		case Sequence:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode Sequence")
			}
			p.sequence = &v
		case FragIndex:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode FragIndex")
			}
			p.fragIndex = &v
		case FragCount:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode FragCount")
			}
			p.fragCount = &v
		case PitToken:
			val := elem.Value()
			p.pitToken = make([]byte, len(val))
			copy(p.pitToken, val)
		case IncomingFaceID:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode IncomingFaceID")
			}
			p.incomingFaceID = &v
		case NextHopFaceID:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode NextHopFaceID")
			}
			p.nextHopFaceID = &v
		case CachePolicy:
			if e := elem.Parse(); e != nil {
				return nil, errors.New("unable to decode CachePolicy")
			}
			cachePolicyType := elem.Find(CachePolicyType)
			if cachePolicyType == nil {
				return nil, errors.New("CachePolicy element does not contain CachePolicyType")
			}
			v, e := tlv.DecodeNNIBlock(cachePolicyType)
			if e != nil {
				return nil, errors.New("unable to decode CachePolicyType")
			}
			p.cachePolicyType = &v
		case CongestionMark:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode CongestionMark")
			}
			p.congestionMark = &v
		case Ack:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode Ack")
			}
			p.acks = append(p.acks, v)
		case TxSequence:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode TxSequence")
			}
			p.txSequence = &v
		case NonDiscovery:
			p.nonDiscovery = true
		case PrefixAnnouncement:
			if e := elem.Parse(); e != nil {
				return nil, errors.New("unable to parse PrefixAnnouncement: " + e.Error())
			}
			data := elem.Find(tlv.Data)
			if data == nil {
				return nil, errors.New("PrefixAnnouncement does not contain Data")
			}
			pa, e := ndn.DecodeData(data, false)
			if e != nil {
				return nil, fmt.Errorf("unable to decode PrefixAnnouncement: %w", e)
			}
			p.prefixAnnouncement = pa
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

// DecodePacket returns an NDNLPv2 frame decoded from the wire.
func DecodePacket(wire *tlv.Block) (*Packet, error) {
	if wire == nil {
		return nil, errors.New("wire is unset")
	}

	p := &Packet{}

	// If type is not LpPacket, then this is a "bare" packet.
	if wire.Type() != LpPacket {
		encodedFragment, err := wire.Wire()
		if err != nil {
			return nil, errors.New("unable to encode bare fragment: " + err.Error())
		}
		// copy encoded fragment
		p.fragment = make([]byte, len(encodedFragment))
		copy(p.fragment, encodedFragment)
		return p, nil
	}

	p.wire = wire
	if e := p.wire.Parse(); e != nil {
		return nil, errors.New("unable to decode LpPacket")
	}
	for _, elem := range p.wire.Subelements() {
		switch elem.Type() {
		case Fragment:
			val := elem.Value()
			p.fragment = make([]byte, len(val))
			copy(p.fragment, val)
		case Sequence:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode Sequence")
			}
			p.sequence = &v
		case FragIndex:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode FragIndex")
			}
			p.fragIndex = &v
		case FragCount:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode FragCount")
			}
			p.fragCount = &v
		case PitToken:
			val := elem.Value()
			p.pitToken = make([]byte, len(val))
			copy(p.pitToken, val)
		case IncomingFaceID:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode IncomingFaceID")
			}
			p.incomingFaceID = &v
		case NextHopFaceID:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode NextHopFaceID")
			}
			p.nextHopFaceID = &v
		case CachePolicy:
			if e := elem.Parse(); e != nil {
				return nil, errors.New("unable to decode CachePolicy")
			}
			cachePolicyType := elem.Find(CachePolicyType)
			if cachePolicyType == nil {
				return nil, errors.New("CachePolicy element does not contain CachePolicyType")
			}
			v, e := tlv.DecodeNNIBlock(cachePolicyType)
			if e != nil {
				return nil, errors.New("unable to decode CachePolicyType")
			}
			p.cachePolicyType = &v
		case CongestionMark:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode CongestionMark")
			}
			p.congestionMark = &v
		case Ack:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode Ack")
			}
			p.acks = append(p.acks, v)
		case TxSequence:
			v, e := tlv.DecodeNNIBlock(elem)
			if e != nil {
				return nil, errors.New("unable to decode TxSequence")
			}
			p.txSequence = &v
		case NonDiscovery:
			p.nonDiscovery = true
		case PrefixAnnouncement:
			if e := elem.Parse(); e != nil {
				return nil, errors.New("unable to parse PrefixAnnouncement: " + e.Error())
			}
			data := elem.Find(tlv.Data)
			if data == nil {
				return nil, errors.New("PrefixAnnouncement does not contain Data")
			}
			pa, e := ndn.DecodeData(data, false)
			if e != nil {
				return nil, fmt.Errorf("unable to decode PrefixAnnouncement: %w", e)
			}
			p.prefixAnnouncement = pa
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
	return (p.sequence == nil && p.fragIndex == nil && p.fragCount == nil &&
		len(p.pitToken) == 0 && p.nextHopFaceID == nil && p.incomingFaceID == nil &&
		p.cachePolicyType == nil && p.congestionMark == nil && p.txSequence == nil &&
		len(p.acks) == 0 && !p.nonDiscovery && p.prefixAnnouncement == nil && p.prefixAnnouncement != nil)
}

// IsIdle returns whether the LpPacket is an "IDLE" frame and does not contain a fragment.
func (p *Packet) IsIdle() bool {
	return len(p.fragment) == 0
}

// Sequence returns the Sequence of the LpPacket or nil if it is unset.
func (p *Packet) Sequence() *uint64 {
	return cloneUint64Ptr(p.sequence)
}

// SetSequence sets the Sequence of the LpPacket.
func (p *Packet) SetSequence(sequence uint64) {
	p.sequence = &sequence
	p.wire = nil
}

// FragIndex returns the FragIndex of the LpPacket or nil if it is unset.
func (p *Packet) FragIndex() *uint64 {
	return cloneUint64Ptr(p.fragIndex)
}

// SetFragIndex sets the FragIndex of the LpPacket.
func (p *Packet) SetFragIndex(fragIndex uint64) {
	p.fragIndex = &fragIndex
	p.wire = nil
}

// FragCount returns the FragCount of the LpPacket or nil if it is unset.
func (p *Packet) FragCount() *uint64 {
	return cloneUint64Ptr(p.fragCount)
}

// SetFragCount sets the FragCount of the LpPacket.
func (p *Packet) SetFragCount(fragCount uint64) {
	p.fragCount = &fragCount
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
	return cloneUint64Ptr(p.nextHopFaceID)
}

// SetNextHopFaceID sets the NextHopFaceId of the LpPacket.
func (p *Packet) SetNextHopFaceID(nextHopFaceID uint64) {
	p.nextHopFaceID = &nextHopFaceID
	p.wire = nil
}

// IncomingFaceID returns the IncomingFaceId of the LpPacket or nil if it is unset.
func (p *Packet) IncomingFaceID() *uint64 {
	return cloneUint64Ptr(p.incomingFaceID)
}

// SetIncomingFaceID sets the IncomingFaceId of the LpPacket.
func (p *Packet) SetIncomingFaceID(incomingFaceID uint64) {
	p.incomingFaceID = &incomingFaceID
	p.wire = nil
}

// CachePolicyType returns the CachePolicyType of the LpPacket or nil if it is unset.
func (p *Packet) CachePolicyType() *uint64 {
	return cloneUint64Ptr(p.cachePolicyType)
}

// SetCachePolicytype sets the CachePolicyType of the LpPacket.
func (p *Packet) SetCachePolicytype(cachePolicyType uint64) {
	p.cachePolicyType = &cachePolicyType
	p.wire = nil
}

// CongestionMark returns the CongestionMark of the LpPacket or nil if it is unset.
func (p *Packet) CongestionMark() *uint64 {
	return cloneUint64Ptr(p.congestionMark)
}

// SetCongestionMark sets the CongestionMark of the LpPacket.
func (p *Packet) SetCongestionMark(congestionMark uint64) {
	p.congestionMark = &congestionMark
	p.wire = nil
}

// TxSequence returns the TxSequence of the LpPacket or nil if it is unset.
func (p *Packet) TxSequence() *uint64 {
	return cloneUint64Ptr(p.txSequence)
}

// SetTxSequence sets the TxSequence of the LpPacket.
func (p *Packet) SetTxSequence(txSequence uint64) {
	p.txSequence = &txSequence
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
	return p.prefixAnnouncement
}

// SetPrefixAnnouncement sets the PrefixAnnouncement field of the LpPacket.
func (p *Packet) SetPrefixAnnouncement(prefixAnnouncement *ndn.Data) {
	p.prefixAnnouncement = prefixAnnouncement
	p.wire = nil
}

// Fragment returns the Fragment field of the LpPacket or nil if it is unset.
func (p *Packet) Fragment() []byte {
	fragment := make([]byte, len(p.fragment))
	copy(fragment, p.fragment)
	return fragment
}

func (p *Packet) FragmentNoCopy() []byte {
	return p.fragment
}

// SetFragment sets the Fragment field of the LpPacket.
func (p *Packet) SetFragment(fragment []byte) {
	p.fragment = make([]byte, len(fragment))
	copy(p.fragment, fragment)
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

		// IncomingFaceID
		if p.incomingFaceID != nil {
			p.wire.Append(tlv.EncodeNNIBlock(IncomingFaceID, *p.incomingFaceID))
		}

		// NextHopFaceID
		if p.nextHopFaceID != nil {
			p.wire.Append(tlv.EncodeNNIBlock(NextHopFaceID, *p.nextHopFaceID))
		}

		// CachePolicyType
		if p.cachePolicyType != nil {
			cachePolicyTypeBlock := tlv.EncodeNNIBlock(CachePolicyType, *p.cachePolicyType)
			cachePolicyTypeBlockWire, err := cachePolicyTypeBlock.Wire()
			if err != nil {
				return nil, errors.New("unable to encode CachePolicyType")
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
				return nil, errors.New("unable to encode PrefixAnnouncement: " + err.Error())
			}
			prefixAnnouncementWire, err := prefixAnnouncementBlock.Wire()
			if err != nil {
				return nil, errors.New("unable to encode PrefixAnnouncement: " + err.Error())
			}
			p.wire.Append(tlv.NewBlock(PrefixAnnouncement, prefixAnnouncementWire))
		}

		// Fragment
		if len(p.fragment) > 0 {
			p.wire.Append(tlv.NewBlock(Fragment, p.fragment))
		}
	}
	p.wire.Wire()
	return p.wire, nil
}
