/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_test

import (
	"testing"
	"time"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/stretchr/testify/assert"
)

func TestMetaInfoNew(t *testing.T) {
	m := ndn.NewMetaInfo()
	assert.NotNil(t, m)
	assert.Nil(t, m.ContentType())
	assert.Nil(t, m.FreshnessPeriod())
	assert.Nil(t, m.FinalBlockID())
}

func TestMetaInfoDecode(t *testing.T) {
	b := tlv.NewBlock(tlv.MetaInfo, []byte{tlv.ContentType, 0x01, 0x04,
		tlv.FreshnessPeriod, 0x02, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e})
	m, err := ndn.DecodeMetaInfo(b)
	assert.NotNil(t, m)
	assert.NoError(t, err)
	assert.True(t, m.HasWire())

	assert.NotNil(t, m.ContentType())
	assert.Equal(t, uint64(4), *m.ContentType())
	assert.NotNil(t, m.FreshnessPeriod())
	assert.Equal(t, time.Duration(5000)*time.Millisecond, *m.FreshnessPeriod())
	assert.NotNil(t, m.FinalBlockID())
	assert.Equal(t, "ndn", m.FinalBlockID().String())
	assert.True(t, m.HasWire())
}
func TestMetaInfoEncode(t *testing.T) {
	m := ndn.NewMetaInfo()
	m.SetContentType(0x04)
	m.SetFreshnessPeriod(time.Duration(5000) * time.Millisecond)
	m.SetFinalBlockID(ndn.NewGenericNameComponent([]byte("ndn")))

	wire := []byte{tlv.MetaInfo, 0x0e,
		tlv.ContentType, 0x01, 0x04,
		tlv.FreshnessPeriod, 0x02, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e}

	assert.False(t, m.HasWire())
	encodedBlock, err := m.Encode()
	assert.NotNil(t, encodedBlock)
	assert.NoError(t, err)
	assert.True(t, m.HasWire())

	encodedWire, err := encodedBlock.Wire()
	assert.NotNil(t, encodedWire)
	assert.NoError(t, err)
	assert.ElementsMatch(t, wire, encodedWire)
}

func TestMetaInfoWire(t *testing.T) {
	m := ndn.NewMetaInfo()
	assert.NotNil(t, m)
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())

	m.SetContentType(0x04)
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())
	m.UnsetContentType()
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())

	m.SetFreshnessPeriod(time.Duration(5000) * time.Millisecond)
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())
	m.UnsetFreshnessPeriod()
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())

	m.SetFinalBlockID(ndn.NewGenericNameComponent([]byte("ndn")))
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())
	m.UnsetFinalBlockID()
	assert.False(t, m.HasWire())
	m.Encode()
	assert.True(t, m.HasWire())
}
