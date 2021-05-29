/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

import (
	"bytes"
	"math"

	"github.com/named-data/YaNFD/ndn/util"
)

// MaxNDNPacketSize is the maximum allowed NDN packet size
const MaxNDNPacketSize = 8800

// Block contains an encoded block.
type Block struct {
	// Contents
	tlvType     uint32
	value       []byte
	subelements []*Block

	// Encoding
	wire []byte
}

///////////////
// Constructors
///////////////

// NewEmptyBlock creates an empty encoded block.
func NewEmptyBlock(tlvType uint32) *Block {
	var block Block
	block.tlvType = tlvType
	return &block
}

// NewBlock creates a block containing the specified type and value.
func NewBlock(tlvType uint32, value []byte) *Block {
	var block Block
	block.tlvType = tlvType
	block.value = value
	// copy(block.value, value)
	return &block
}

//////////
// Getters
//////////

// Type returns the type of the block.
func (b *Block) Type() uint32 {
	return b.tlvType
}

// Value returns the value contained in the block.
func (b *Block) Value() []byte {
	return b.value
}

// Subelements returns the sub-elements of the block.
func (b *Block) Subelements() []*Block {
	return b.subelements
}

//////////
// Setters
//////////

// SetType sets the TLV type of the block.
func (b *Block) SetType(tlvType uint32) {
	if b.tlvType != tlvType {
		b.tlvType = tlvType
		b.wire = []byte{}
	}
}

// SetValue sets the value of the block.
func (b *Block) SetValue(value []byte) {
	if !bytes.Equal(b.value, value) {
		b.value = value
		// copy(b.value, value)
		b.wire = []byte{}
	}
}

//////////////
// Subelements
//////////////

// Append appends a subelement onto the end of the block's value.
func (b *Block) Append(block *Block) {
	b.subelements = append(b.subelements, block)
	b.wire = []byte{}
}

// Clear erases all subelements of the block.
func (b *Block) Clear() {
	if len(b.subelements) > 0 {
		b.subelements = []*Block{}
		b.wire = []byte{}
	}
}

// DeepCopy creates a deep copy of the block.
func (b *Block) DeepCopy() *Block {
	copyB := new(Block)
	copyB.tlvType = b.tlvType
	copyB.value = make([]byte, len(b.value))
	copy(copyB.value, b.value)
	copyB.subelements = make([]*Block, 0, len(b.subelements))
	for _, subelem := range b.subelements {
		copyB.subelements = append(copyB.subelements, subelem.DeepCopy())
	}
	copyB.wire = make([]byte, len(b.wire))
	copy(copyB.wire, b.wire)
	return copyB
}

// Encode encodes all subelements into the block's value.
func (b *Block) Encode() error {
	if len(b.subelements) == 0 {
		// Take no action, but is not an error
		return nil
	}

	b.value = []byte{}
	for _, elem := range b.subelements {
		elemWire, err := elem.Wire()
		if err != nil {
			b.value = []byte{}
			return err
		}
		b.value = append(b.value, elemWire...)
	}

	b.subelements = []*Block{}
	return nil
}

// Erase erases the first subelement of the specified type and returns whether an element was removed.
func (b *Block) Erase(tlvType uint32) bool {
	indexToRemove := -1
	for i, elem := range b.subelements {
		if elem.Type() == tlvType {
			indexToRemove = i
			break
		}
	}

	if indexToRemove != -1 {
		copy(b.subelements[indexToRemove:], b.subelements[indexToRemove+1:])
		b.subelements = b.subelements[:len(b.subelements)-1]
		b.wire = []byte{}
	}

	return indexToRemove != -1
}

// EraseAll erases all subelements of the specified type and returns the count of elements removed.
func (b *Block) EraseAll(tlvType uint32) int {
	numErased := 0
	shouldContinue := true
	for shouldContinue {
		if b.Erase(tlvType) {
			numErased++
		} else {
			shouldContinue = false
		}
	}
	return numErased
}

// Find returns the first subelement of the specified type, or nil if none exists.
func (b *Block) Find(tlvType uint32) *Block {
	for _, elem := range b.subelements {
		if elem.Type() == tlvType {
			return elem
		}
	}
	return nil
}

// Insert inserts the subelement in order of ascending TLV type, after any subelements of the same TLV type. Note that this assumes subelements are ordered by increasing TLV type.
func (b *Block) Insert(in *Block) {
	block := in
	if len(b.subelements) == 0 {
		b.subelements = []*Block{block}
	} else if b.subelements[0].Type() > block.Type() {
		b.subelements = append([]*Block{block}, b.subelements...)
	} else if b.subelements[len(b.subelements)-1].Type() <= block.Type() {
		b.subelements = append(b.subelements, block)
	} else {
		precedingElem := 0
		for i, elem := range b.subelements {
			if elem.Type() > block.Type() {
				precedingElem = i - 1
				break
			}
		}
		b.subelements = append(b.subelements[:precedingElem+1], append([]*Block{block}, b.subelements[precedingElem+1:]...)...)
	}

	b.wire = []byte{}
}

// Parse parses the block value into subelements, if possible.
func (b *Block) Parse() error {
	startPos := uint64(0)
	b.subelements = []*Block{}
	for startPos < uint64(len(b.value)) {
		block, blockLen, err := DecodeBlock(b.value[startPos:])
		if err != nil {
			return err
		}
		b.subelements = append(b.subelements, block)
		startPos += blockLen
	}
	return nil
}

////////////////////
// Encoding/Decoding
////////////////////

func varSize(in uint64) uint64 {
	switch {
	case in <= 0xFC:
		return 1
	case in <= 0xFFFF:
		return 3
	case in <= 0xFFFFFFFF:
		return 5
	default:
		return 9
	}
}

// Wire returns the wire-encoded block.
func (b *Block) Wire() ([]byte, error) {
	if len(b.wire) == 0 {
		// There is still unnecessary copying, but better than the original one.
		l := uint64(0)
		if len(b.subelements) > 0 {
			for _, elem := range b.subelements {
				elemWire, err := elem.Wire()
				if err != nil {
					return []byte{}, err
				}
				l += uint64(len(elemWire))
			}
		} else {
			l = uint64(len(b.value))
		}

		// Encode type, length, and value into wire
		wireSz := varSize(uint64(b.tlvType)) + l + varSize(l)
		b.wire = make([]byte, 0, wireSz)
		encodedType := EncodeVarNum(uint64(b.tlvType))
		b.wire = append(b.wire, encodedType...)
		encodedLength := EncodeVarNum(l)
		b.wire = append(b.wire, encodedLength...)

		if len(b.subelements) > 0 {
			// Wire encode subelements
			for _, elem := range b.subelements {
				b.wire = append(b.wire, elem.wire...)
			}
		} else {
			b.wire = append(b.wire, b.value...)
		}
	}

	return b.wire, nil
}

// HasWire returns whether the block has a valid wire encoding.
func (b *Block) HasWire() bool {
	return len(b.wire) > 0
}

// Size returns the size of the wire.
func (b *Block) Size() int {
	return len(b.wire)
}

// Reset clears the encoded wire buffer, value, and subelements of the block.
func (b *Block) Reset() {
	b.wire = []byte{}
	b.value = []byte{}
	b.subelements = []*Block{}
}

// DecodeBlock decodes a block from the wire.
func DecodeBlock(wire []byte) (*Block, uint64, error) {
	b := new(Block)

	// Decode TLV type
	tlvType, tlvTypeLen, err := DecodeVarNum(wire)
	if err != nil {
		return nil, 0, err
	}
	if tlvType > math.MaxUint32 {
		return nil, 0, util.ErrOutOfRange
	}
	b.tlvType = uint32(tlvType)

	// Decode TLV length (we don't store this because it's implicit from value slice length)
	if tlvTypeLen == len(wire) {
		return nil, 0, ErrMissingLength
	}
	tlvLength, tlvLengthLen, err := DecodeVarNum(wire[tlvTypeLen:])
	if err != nil {
		return nil, 0, err
	}

	// Decode TLV value
	if uint64(len(wire)) < uint64(tlvTypeLen)+uint64(tlvLengthLen)+tlvLength {
		return nil, 0, ErrBufferTooShort
	}
	// b.value = make([]byte, tlvLength)
	b.value = wire[tlvTypeLen+tlvLengthLen : uint64(tlvTypeLen)+uint64(tlvLengthLen)+tlvLength]

	// Add wire
	// b.wire = make([]byte, uint64(tlvTypeLen)+uint64(tlvLengthLen)+tlvLength)
	b.wire = wire[:uint64(tlvTypeLen)+uint64(tlvLengthLen)+tlvLength]

	return b, uint64(tlvTypeLen) + uint64(tlvLengthLen) + tlvLength, nil
}
