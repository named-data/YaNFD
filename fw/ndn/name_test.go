/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_test

import (
	"testing"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/stretchr/testify/assert"
)

func TestNameCreate(t *testing.T) {
	n := ndn.NewName()
	assert.NotNil(t, n)

	encoded, err := n.Encode().Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, encoded, []byte{0x07, 0x00})

	assert.Equal(t, "/", n.String())
}

func TestNameDecode(t *testing.T) {
	n, err := ndn.DecodeName(nil)
	assert.Nil(t, n)
	assert.Error(t, err)

	n, err = ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x00}))
	assert.Nil(t, n)
	assert.Error(t, err)

	n, err = ndn.DecodeName(tlv.NewBlock(0x08, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e}))
	assert.Nil(t, n)
	assert.Error(t, err)

	n, err = ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e}))
	assert.NotNil(t, n)
	assert.NoError(t, err)

	assert.Equal(t, 2, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, []byte{0x67, 0x6f}, n.At(0).Value())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, []byte{0x6e, 0x64, 0x6e}, n.At(1).Value())
	assert.Equal(t, "ndn", n.At(1).String())

	assert.Equal(t, "/go/ndn", n.String())
}

func TestNameDecodeUnknownComponent(t *testing.T) {
	n, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0xDD, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e}))
	assert.NotNil(t, n)
	assert.NoError(t, err)

	assert.Equal(t, 2, n.Size())
	assert.Equal(t, uint16(0xDD), n.At(0).Type())
	assert.Equal(t, []byte{0x67, 0x6f}, n.At(0).Value())
	assert.Equal(t, "221=go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, []byte{0x6e, 0x64, 0x6e}, n.At(1).Value())
	assert.Equal(t, "ndn", n.At(1).String())

	assert.Equal(t, "/221=go/ndn", n.String())
}

func TestNameComponents(t *testing.T) {
	n := new(ndn.Name)

	goComponent := ndn.NewGenericNameComponent([]byte("go"))
	assert.NotNil(t, goComponent)
	n.Append(goComponent)
	assert.Equal(t, 1, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())

	ndnComponent := ndn.NewGenericNameComponent([]byte("ndn"))
	assert.NotNil(t, ndnComponent)
	n.Append(ndnComponent)
	assert.Equal(t, 2, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "ndn", n.At(1).String())

	n.Append(goComponent)
	assert.Equal(t, 3, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "ndn", n.At(1).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(2).Type())
	assert.Equal(t, "go", n.At(2).String())

	// Test replacing
	segComponent := ndn.NewSegmentNameComponent(27)
	assert.NotNil(t, segComponent)
	assert.NoError(t, n.Set(2, segComponent))
	assert.Equal(t, 3, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "ndn", n.At(1).String())
	assert.Equal(t, uint16(tlv.SegmentNameComponent), n.At(2).Type())
	assert.Equal(t, "seg=27", n.At(2).String())

	// Test insertion at arbitrary index
	assert.NoError(t, n.Insert(2, goComponent))
	assert.Equal(t, 4, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "ndn", n.At(1).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(2).Type())
	assert.Equal(t, "go", n.At(2).String())
	assert.Equal(t, uint16(tlv.SegmentNameComponent), n.At(3).Type())
	assert.Equal(t, "seg=27", n.At(3).String())

	// Test finding first name component of type
	index, matching := n.Find(tlv.SegmentNameComponent)
	assert.Equal(t, 3, index)
	assert.NotNil(t, matching)
	assert.Equal(t, uint16(tlv.SegmentNameComponent), matching.Type())
	index, matching = n.Find(tlv.ImplicitSha256DigestComponent)
	assert.Equal(t, -1, index)
	assert.Nil(t, matching)

	// Test removal
	assert.NoError(t, n.Erase(1))
	assert.Equal(t, 3, n.Size())
	assert.Nil(t, n.At(3))
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "go", n.At(1).String())
	assert.Equal(t, uint16(tlv.SegmentNameComponent), n.At(2).Type())
	assert.Equal(t, "seg=27", n.At(2).String())

	// Test clearing
	n.Clear()
	assert.Equal(t, 0, n.Size())

	// Test out of bounds access
	assert.Nil(t, n.At(0))
}

func TestNameComparison(t *testing.T) {
	n, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e, 0x21, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAA}))
	assert.NotNil(t, n)
	assert.NoError(t, err)
	assert.Equal(t, 3, n.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "ndn", n.At(1).String())
	assert.Equal(t, uint16(tlv.SegmentNameComponent), n.At(2).Type())
	assert.Equal(t, "seg=170", n.At(2).String())

	prefix := n.Prefix(2)
	assert.NotNil(t, prefix)
	assert.Equal(t, 2, prefix.Size())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(0).Type())
	assert.Equal(t, "go", n.At(0).String())
	assert.Equal(t, uint16(tlv.GenericNameComponent), n.At(1).Type())
	assert.Equal(t, "ndn", n.At(1).String())

	// Comparisons
	assert.True(t, n.Equals(n))
	assert.True(t, prefix.Equals(prefix))
	assert.False(t, n.Equals(prefix))
	assert.False(t, prefix.Equals(n))
	assert.True(t, prefix.PrefixOf(n))
	assert.False(t, n.PrefixOf(prefix))

	nNdnGo, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x03, 0x6e, 0x64, 0x6e, 0x08, 0x02, 0x67, 0x6f, 0x21, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAA}))
	assert.NotNil(t, n)
	assert.NoError(t, err)
	assert.False(t, n.Equals(nNdnGo))
	assert.False(t, nNdnGo.Equals(n))

	n1 := n.DeepCopy()
	goComponent := ndn.NewGenericNameComponent([]byte("go"))
	assert.NotNil(t, goComponent)
	assert.NoError(t, n1.Set(1, goComponent))
	assert.False(t, n.PrefixOf(n1))
	assert.False(t, n1.PrefixOf(n))
	assert.False(t, n.Equals(n1))
	assert.False(t, n1.Equals(n))
}

func TestNameEncode(t *testing.T) {
	n, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e, 0x21, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAA}))
	assert.NotNil(t, n)
	assert.NoError(t, err)
	assert.True(t, n.HasWire())
	b := n.Encode()
	wire, err := b.Wire()
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x07, 0x13, 0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e, 0x21, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAA}, wire)

	goComponent := ndn.NewGenericNameComponent([]byte("go"))
	assert.NotNil(t, goComponent)
	assert.NoError(t, n.Set(1, goComponent))
	assert.False(t, n.HasWire())

	b = n.Encode()
	assert.NotNil(t, b)
	assert.True(t, n.HasWire())
	wire, err = b.Wire()
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x07, 0x12, 0x08, 0x02, 0x67, 0x6f, 0x08, 0x02, 0x67, 0x6f, 0x21, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAA}, wire)
}

func TestNameCompare(t *testing.T) {
	n1, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e, 0x21, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAA}))
	assert.NotNil(t, n1)
	assert.NoError(t, err)
	n2, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e}))
	assert.NotNil(t, n2)
	assert.NoError(t, err)
	n3, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6f}))
	assert.NotNil(t, n3)
	assert.NoError(t, err)
	n4, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x09, 0x03, 0x6e, 0x64, 0x6e}))
	assert.NotNil(t, n4)
	assert.NoError(t, err)
	n5, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x09, 0x04, 0x6e, 0x64, 0x6e, 0x6e}))
	assert.NotNil(t, n5)
	assert.NoError(t, err)

	// Test when equal
	assert.Equal(t, 0, n1.Compare(n1))

	// Test when prefix
	assert.Equal(t, -1, n2.Compare(n1))
	assert.Equal(t, 1, n1.Compare(n2))

	// Test when type differs
	assert.Equal(t, -1, n2.Compare(n4))
	assert.Equal(t, 1, n4.Compare(n2))

	// Test when component lengths differ
	assert.Equal(t, -1, n4.Compare(n5))
	assert.Equal(t, 1, n5.Compare(n4))

	// Test when component values differ
	assert.Equal(t, -1, n2.Compare(n3))
	assert.Equal(t, 1, n3.Compare(n2))
}

func TestNameEscape(t *testing.T) {
	n1, err := ndn.DecodeName(tlv.NewBlock(0x07, []byte{0x08, 0x02, 0x67, 0x6f, 0x08, 0x03, 0x6e, 0x64, 0x6e, 0x08, 0x08, 0x30, 0x31, 0x32, 0x33, 0x24, 0x30, 0x2e, 0x2a}))
	assert.NotNil(t, n1)
	assert.NoError(t, err)
	assert.Equal(t, "/go/ndn/0123%240.%2a", n1.String())
}
