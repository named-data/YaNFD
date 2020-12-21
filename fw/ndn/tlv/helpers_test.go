/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv_test

import (
	"testing"

	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/stretchr/testify/assert"
)

func TestVarNum(t *testing.T) {
	octet1 := []byte{0x01}
	octet3 := []byte{0xFD, 0x01, 0x02}
	octet5 := []byte{0xFE, 0x01, 0x02, 0x03, 0x04}
	octet9 := []byte{0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	decoded1, length, err := tlv.DecodeVarNum(octet1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x01), decoded1)
	assert.Equal(t, 1, length)

	decoded3, length, err := tlv.DecodeVarNum(octet3)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x0102), decoded3)
	assert.Equal(t, 3, length)

	decoded5, length, err := tlv.DecodeVarNum(octet5)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x01020304), decoded5)
	assert.Equal(t, 5, length)

	decoded9, length, err := tlv.DecodeVarNum(octet9)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x0102030405060708), decoded9)
	assert.Equal(t, 9, length)

	encoded1 := tlv.EncodeVarNum(0x01)
	assert.ElementsMatch(t, octet1, encoded1)

	encoded3 := tlv.EncodeVarNum(0x0102)
	assert.ElementsMatch(t, octet3, encoded3)

	encoded5 := tlv.EncodeVarNum(0x01020304)
	assert.ElementsMatch(t, octet5, encoded5)

	encoded9 := tlv.EncodeVarNum(0x0102030405060708)
	assert.ElementsMatch(t, octet9, encoded9)
}

func TestVarNumTooShort(t *testing.T) {
	octet1 := []byte{}
	octet3 := []byte{0xFD, 0x01}
	octet5 := []byte{0xFE, 0x01, 0x02, 0x03}
	octet9 := []byte{0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}

	_, _, err := tlv.DecodeVarNum(octet1)
	assert.EqualError(t, err, "Value too short")

	_, _, err = tlv.DecodeVarNum(octet3)
	assert.EqualError(t, err, "Value too short")

	_, _, err = tlv.DecodeVarNum(octet5)
	assert.EqualError(t, err, "Value too short")

	_, _, err = tlv.DecodeVarNum(octet9)
	assert.EqualError(t, err, "Value too short")
}

func TestNNIBlock(t *testing.T) {
	nni := uint64(0x0102030405060708)
	blockType := uint32(0x27)
	nniBlock := tlv.EncodeNNIBlock(blockType, nni)
	nniWire := []byte{0x27, 0x08, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	assert.Equal(t, blockType, nniBlock.Type())
	encodedWire, err := nniBlock.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, nniWire, encodedWire)
}

func TestEncodeNNI(t *testing.T) {
	nni1 := uint64(0x01)
	octet1 := []byte{0x01}
	block1 := tlv.EncodeNNIBlock(0xAA, nni1)
	assert.ElementsMatch(t, octet1, block1.Value())

	nni2 := uint64(0x0102)
	octet2 := []byte{0x01, 0x02}
	block2 := tlv.EncodeNNIBlock(0xAA, nni2)
	assert.ElementsMatch(t, octet2, block2.Value())

	nni3 := uint64(0x010203)
	octet3 := []byte{0x00, 0x01, 0x02, 0x03}
	block3 := tlv.EncodeNNIBlock(0xAA, nni3)
	assert.ElementsMatch(t, octet3, block3.Value())

	nni4 := uint64(0x01020304)
	octet4 := []byte{0x01, 0x02, 0x03, 0x04}
	block4 := tlv.EncodeNNIBlock(0xAA, nni4)
	assert.ElementsMatch(t, octet4, block4.Value())

	nni5 := uint64(0x0102030405)
	octet5 := []byte{0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	block5 := tlv.EncodeNNIBlock(0xAA, nni5)
	assert.ElementsMatch(t, octet5, block5.Value())

	nni6 := uint64(0x010203040506)
	octet6 := []byte{0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	block6 := tlv.EncodeNNIBlock(0xAA, nni6)
	assert.ElementsMatch(t, octet6, block6.Value())

	nni7 := uint64(0x01020304050607)
	octet7 := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	block7 := tlv.EncodeNNIBlock(0xAA, nni7)
	assert.ElementsMatch(t, octet7, block7.Value())

	nni8 := uint64(0x0102030405060708)
	octet8 := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	block8 := tlv.EncodeNNIBlock(0xAA, nni8)
	assert.ElementsMatch(t, octet8, block8.Value())
}
