/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv_test

import (
	"testing"

	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/stretchr/testify/assert"
)

func TestBlockCreateAndEncode(t *testing.T) {
	block := tlv.NewBlock(0x28, []byte{0x01, 0x02, 0x03, 0x04})
	assert.NotNil(t, block)
	assert.Equal(t, uint32(0x28), block.Type())
	assert.ElementsMatch(t, []byte{0x01, 0x02, 0x03, 0x04}, block.Value())
	assert.False(t, block.HasWire())
	encoded, err := block.Wire()
	assert.NoError(t, err)
	assert.Equal(t, 6, block.Size())
	assert.ElementsMatch(t, []byte{0x28, 0x04, 0x01, 0x02, 0x03, 0x04}, encoded)
	assert.True(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.Equal(t, 6, block.Size())
	assert.ElementsMatch(t, []byte{0x28, 0x04, 0x01, 0x02, 0x03, 0x04}, encoded)
	assert.True(t, block.HasWire())

	block = tlv.NewEmptyBlock(0x28)
	assert.NotNil(t, block)
	assert.Equal(t, uint32(0x28), block.Type())
	assert.Equal(t, 0, len(block.Value()))
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.Equal(t, 2, block.Size())
	assert.ElementsMatch(t, []byte{0x28, 0x00}, encoded)

	assert.True(t, block.HasWire())
	block.Reset()
	assert.False(t, block.HasWire())
}

func TestBlockDecode(t *testing.T) {
	block, blockSize, err := tlv.DecodeBlock([]byte{0x28, 0x04, 0x01, 0x02, 0x03, 0x04})
	assert.NotNil(t, block)
	assert.Equal(t, uint64(6), blockSize)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0x28), block.Type())
	assert.True(t, block.HasWire())
	assert.Equal(t, 6, block.Size())
	assert.ElementsMatch(t, []byte{0x01, 0x02, 0x03, 0x04}, block.Value())
	encoded, err := block.Wire()
	assert.NoError(t, err)
	assert.True(t, block.HasWire())
	assert.Equal(t, 6, block.Size())
	assert.ElementsMatch(t, []byte{0x28, 0x04, 0x01, 0x02, 0x03, 0x04}, encoded)
}

func TestBlockSetters(t *testing.T) {
	block := tlv.NewBlock(0x30, []byte{0x01, 0x02, 0x03, 0x04, 0x05})
	assert.NotNil(t, block)
	block.Wire()
	assert.True(t, block.HasWire())

	// Set Type
	block.SetType(0x47)
	assert.False(t, block.HasWire())
	encoded, err := block.Wire()
	assert.NoError(t, err)
	assert.True(t, block.HasWire())
	assert.ElementsMatch(t, []byte{0x47, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05}, encoded)

	// Set Value
	block.SetValue([]byte{0xF0, 0xF1, 0xF2, 0xF3})
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.True(t, block.HasWire())
	assert.ElementsMatch(t, []byte{0x47, 0x04, 0xF0, 0xF1, 0xF2, 0xF3}, encoded)
}

func TestBlockSubelements(t *testing.T) {
	block := tlv.NewEmptyBlock(0x77)
	assert.NotNil(t, block)
	assert.Equal(t, 0, len(block.Subelements()))

	// Append a subelement
	subElemA0 := tlv.NewBlock(0xA0, []byte{0x20})
	block.Wire()
	assert.True(t, block.HasWire())
	block.Append(subElemA0)
	assert.False(t, block.HasWire())
	encoded, err := block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x03, 0xA0, 0x01, 0x20}, encoded)
	assert.Equal(t, 1, len(block.Subelements()))
	assert.Equal(t, uint32(0xA0), block.Subelements()[0].Type())

	// Append another subelement
	subElemC3 := tlv.NewBlock(0xC3, []byte{0x30})
	block.Append(subElemC3)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x06,
		0xA0, 0x01, 0x20,
		0xC3, 0x01, 0x30}, encoded)
	assert.Equal(t, 2, len(block.Subelements()))
	assert.Equal(t, uint32(0xA0), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[1].Type())

	// Insert a subelement in order
	subElemB2 := tlv.NewBlock(0xB2, []byte{0x40})
	block.Insert(subElemB2)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x09,
		0xA0, 0x01, 0x20,
		0xB2, 0x01, 0x40,
		0xC3, 0x01, 0x30}, encoded)
	assert.Equal(t, 3, len(block.Subelements()))
	assert.Equal(t, uint32(0xA0), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xB2), block.Subelements()[1].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[2].Type())

	// Insert another C3 in order
	subElemC3A := tlv.NewBlock(0xC3, []byte{0x60})
	block.Insert(subElemC3A)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x0C,
		0xA0, 0x01, 0x20,
		0xB2, 0x01, 0x40,
		0xC3, 0x01, 0x30,
		0xC3, 0x01, 0x60}, encoded)
	assert.Equal(t, 4, len(block.Subelements()))
	assert.Equal(t, uint32(0xA0), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xB2), block.Subelements()[1].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[2].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[3].Type())

	// Insert another A0 in order
	block.Insert(subElemA0)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x0F,
		0xA0, 0x01, 0x20,
		0xA0, 0x01, 0x20,
		0xB2, 0x01, 0x40,
		0xC3, 0x01, 0x30,
		0xC3, 0x01, 0x60}, encoded)
	assert.Equal(t, 5, len(block.Subelements()))
	assert.Equal(t, uint32(0xA0), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xA0), block.Subelements()[1].Type())
	assert.Equal(t, uint32(0xB2), block.Subelements()[2].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[3].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[4].Type())
	findFirstC3 := block.Find(0xC3)
	assert.NotNil(t, findFirstC3)
	assert.Equal(t, uint32(0xC3), findFirstC3.Type())
	assert.ElementsMatch(t, []byte{0x30}, findFirstC3.Value())
	assert.Nil(t, block.Find(0xF6))

	// Insert 90 in order
	subElem90 := tlv.NewBlock(0x90, []byte{0x50})
	block.Insert(subElem90)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x12,
		0x90, 0x01, 0x50,
		0xA0, 0x01, 0x20,
		0xA0, 0x01, 0x20,
		0xB2, 0x01, 0x40,
		0xC3, 0x01, 0x30,
		0xC3, 0x01, 0x60}, encoded)
	assert.Equal(t, 6, len(block.Subelements()))
	assert.Equal(t, uint32(0x90), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xA0), block.Subelements()[1].Type())
	assert.Equal(t, uint32(0xA0), block.Subelements()[2].Type())
	assert.Equal(t, uint32(0xB2), block.Subelements()[3].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[4].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[5].Type())

	// Remove first C3
	block.Erase(0xC3)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x0F,
		0x90, 0x01, 0x50,
		0xA0, 0x01, 0x20,
		0xA0, 0x01, 0x20,
		0xB2, 0x01, 0x40,
		0xC3, 0x01, 0x60}, encoded)
	assert.Equal(t, 5, len(block.Subelements()))
	assert.Equal(t, uint32(0x90), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xA0), block.Subelements()[1].Type())
	assert.Equal(t, uint32(0xA0), block.Subelements()[2].Type())
	assert.Equal(t, uint32(0xB2), block.Subelements()[3].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[4].Type())

	// Remove all A0's
	block.EraseAll(0xA0)
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x09,
		0x90, 0x01, 0x50,
		0xB2, 0x01, 0x40,
		0xC3, 0x01, 0x60}, encoded)
	assert.Equal(t, 3, len(block.Subelements()))
	assert.Equal(t, uint32(0x90), block.Subelements()[0].Type())
	assert.Equal(t, uint32(0xB2), block.Subelements()[1].Type())
	assert.Equal(t, uint32(0xC3), block.Subelements()[2].Type())

	// Erase all subelements
	block.Clear()
	assert.False(t, block.HasWire())
	encoded, err = block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0x77, 0x00}, encoded)
	assert.Equal(t, 0, len(block.Subelements()))
	assert.Nil(t, block.Find(0xA0))
}

func TestBlockEncodeSubelements(t *testing.T) {
	block := tlv.NewEmptyBlock(0xAA)
	block.Append(tlv.NewBlock(0xBB, []byte{0x01}))
	block.Append(tlv.NewBlock(0xCC, []byte{0x02}))
	blockDD := tlv.NewEmptyBlock(0xDD)
	blockDD.Append(tlv.NewBlock(0xEE, []byte{0x03}))
	block.Append(blockDD)

	// Encode
	assert.NoError(t, block.Encode())
	assert.Equal(t, 0, len(block.Subelements()))
	assert.ElementsMatch(t, []byte{0xBB, 0x01, 0x01, 0xCC, 0x01, 0x02, 0xDD, 0x03, 0xEE, 0x01, 0x03}, block.Value())
	encoded, err := block.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []byte{0xAA, 0x0B, 0xBB, 0x01, 0x01, 0xCC, 0x01, 0x02, 0xDD, 0x03, 0xEE, 0x01, 0x03}, encoded)
}

func TestBlockDecodeSubelements(t *testing.T) {
	wire := []byte{0xAA, 0x0B, 0xBB, 0x01, 0x01, 0xCC, 0x01, 0x02, 0xDD, 0x03, 0xEE, 0x01, 0x03}
	block, _, err := tlv.DecodeBlock(wire)
	assert.NotNil(t, block)
	assert.NoError(t, err)

	// Parse
	assert.NoError(t, block.Parse())
	assert.Equal(t, 3, len(block.Subelements()))
	assert.Equal(t, uint32(0xBB), block.Subelements()[0].Type())
	assert.Equal(t, []byte{0x01}, block.Subelements()[0].Value())
	assert.Equal(t, uint32(0xCC), block.Subelements()[1].Type())
	assert.Equal(t, []byte{0x02}, block.Subelements()[1].Value())
	assert.Equal(t, uint32(0xDD), block.Subelements()[2].Type())
	assert.Equal(t, []byte{0xEE, 0x01, 0x03}, block.Subelements()[2].Value())
}

func TestBlockDeepCopy(t *testing.T) {
	block := tlv.NewEmptyBlock(0xCC)
	assert.NotNil(t, block)
	block.Append(tlv.NewEmptyBlock(0xAA))
	block.Append(tlv.NewEmptyBlock(0xBB))
	encodedBlock, _ := block.Wire()

	copyBlock := block.DeepCopy()
	assert.NotNil(t, copyBlock)
	encodedCopyBlock, _ := copyBlock.Wire()
	assert.NotSame(t, &block, &copyBlock)
	assert.NotSame(t, &(block.Subelements()[0]), &(copyBlock.Subelements()[0]))
	assert.NotSame(t, &(block.Subelements()[1]), &(copyBlock.Subelements()[1]))
	assert.NotSame(t, encodedBlock, encodedCopyBlock)
}
