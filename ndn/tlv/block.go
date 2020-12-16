/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

import (
	"bytes"
	"math"

	"github.com/eric135/YaNFD/ndn/util"
)

// Block contains an encoded block.
type Block struct {
	// Contents
	tlvType     uint32
	value       []byte
	subelements []*Block

	// Encoding
	wire    []byte
	hasWire bool
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
	block.value = make([]byte, len(value))
	copy(block.value, value)
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
		b.hasWire = false
	}
}

// SetValue sets the value of the block.
func (b *Block) SetValue(value []byte) {
	if !bytes.Equal(b.value, value) {
		b.value = make([]byte, len(value))
		copy(b.value, value)
		b.hasWire = false
	}
}

//////////////
// Subelements
//////////////

// Append appends a subelement onto the end of the block's value.
func (b *Block) Append(block *Block) {
	b.subelements = append(b.subelements, block.DeepCopy())
	b.hasWire = false
}

// Clear erases all subelements of the block.
func (b *Block) Clear() {
	if len(b.subelements) > 0 {
		b.subelements = []*Block{}
		b.hasWire = false
	}
}

// DeepCopy creates a deep copy of the block.
func (b *Block) DeepCopy() *Block {
	copyB := *b
	copyB.value = make([]byte, len(b.value))
	copy(copyB.value, b.value)
	copyB.subelements = make([]*Block, 0, len(b.subelements))
	for _, subelem := range b.subelements {
		copyB.subelements = append(copyB.subelements, subelem.DeepCopy())
	}
	// Reset wire
	copyB.wire = make([]byte, len(b.wire))
	copy(copyB.wire, b.wire)
	copyB.hasWire = b.hasWire
	return &copyB
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
		b.hasWire = false
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
			b.hasWire = false
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
	block := in.DeepCopy()
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

	b.hasWire = false
}

// Parse parses the block value into subelements, if possible.
func (b *Block) Parse() bool {
	startPos := uint64(0)
	b.subelements = []*Block{}
	for startPos < uint64(len(b.value)) {
		block, blockLen, err := DecodeBlock(b.value[startPos:])
		if err != nil {
			return false
		}
		b.subelements = append(b.subelements, block)
		startPos += blockLen
	}
	b.value = []byte{}
	return true
}

////////////////////
// Encoding/Decoding
////////////////////

// Wire returns the wire-encoded block.
func (b *Block) Wire() ([]byte, error) {
	if b.hasWire {
		return b.wire, nil
	}
	b.wire = []byte{}

	// Encode type, length, and value into wire
	encodedType := EncodeVarNum(uint64(b.tlvType))
	var buf bytes.Buffer
	if len(b.subelements) > 0 {
		// Wire encode subelements
		var elemSize uint64
		for _, elem := range b.subelements {
			elemWire, err := elem.Wire()
			if err != nil {
				return b.wire, err
			}
			elemSize += uint64(len(elemWire))
		}
		encodedLength := EncodeVarNum(elemSize)

		buf.Grow(len(encodedType) + len(encodedLength) + int(elemSize))
		buf.Write(encodedType)
		buf.Write(encodedLength)
		for _, elem := range b.subelements {
			elemWire, err := elem.Wire()
			if err != nil {
				b.wire = []byte{}
				return b.wire, nil
			}
			buf.Write(elemWire)
		}
	} else {
		encodedLength := EncodeVarNum(uint64(len(b.value)))
		buf.Grow(len(encodedType) + len(encodedLength) + len(b.value))
		buf.Write(encodedType)
		buf.Write(encodedLength)
		buf.Write(b.value)
	}

	b.wire = buf.Bytes()
	b.hasWire = true
	return b.wire, nil
}

// HasWire returns whether the block has a valid wire encoding.
func (b *Block) HasWire() bool {
	return b.hasWire
}

// Size returns the size of the wire.
func (b *Block) Size() int {
	return len(b.wire)
}

// Reset clears the encoded wire buffer, value, and subelements of the block.
func (b *Block) Reset() {
	b.hasWire = false
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
	b.value = make([]byte, tlvLength)
	copy(b.value, wire[tlvTypeLen+tlvLengthLen:uint64(tlvTypeLen)+uint64(tlvLengthLen)+tlvLength])

	// Add wire
	b.wire = make([]byte, uint64(tlvTypeLen)+uint64(tlvLengthLen)+tlvLength)
	copy(b.wire, wire)
	b.hasWire = true

	return b, uint64(tlvTypeLen) + uint64(tlvLengthLen) + tlvLength, nil
}
