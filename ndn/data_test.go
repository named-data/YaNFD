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

func TestDataNew(t *testing.T) {
	name, err := ndn.NameFromString("/go/ndn/2020")
	assert.NotNil(t, name)
	assert.NoError(t, err)
	d := ndn.NewData(name, []byte{0x01, 0x02, 0x03, 0x04})
	assert.NotNil(t, d)
	assert.Equal(t, "/go/ndn/2020", d.Name().String())
	assert.NotNil(t, d.MetaInfo())
	assert.ElementsMatch(t, []byte{0x01, 0x02, 0x03, 0x04}, d.Content())
}

func TestDataDecode(t *testing.T) {
	b := tlv.NewBlock(tlv.Data, []byte{
		tlv.Name, 0x0f, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.GenericNameComponent, 0x04, 0x32, 0x30, 0x32, 0x30,
		tlv.MetaInfo, 0x1b,
		tlv.ContentType, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
		tlv.FreshnessPeriod, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e,
		tlv.Content, 0x04, 0x01, 0x02, 0x03, 0x04})
	d, err := ndn.DecodeData(b)
	assert.NotNil(t, d)
	assert.NoError(t, err)
	assert.True(t, d.HasWire())

	assert.Equal(t, "/go/ndn/2020", d.Name().String())
	assert.NotNil(t, d.MetaInfo())
	assert.NotNil(t, d.MetaInfo().ContentType())
	assert.Equal(t, uint64(4), *d.MetaInfo().ContentType())
	assert.NotNil(t, d.MetaInfo().FreshnessPeriod())
	assert.Equal(t, time.Duration(5000)*time.Millisecond, *d.MetaInfo().FreshnessPeriod())
	assert.NotNil(t, d.MetaInfo().FinalBlockID())
	assert.Equal(t, "ndn", d.MetaInfo().FinalBlockID().String())
	assert.ElementsMatch(t, []byte{0x01, 0x02, 0x03, 0x04}, d.Content())
	assert.True(t, d.HasWire())
}

func TestDataEncode(t *testing.T) {
	name, err := ndn.NameFromString("/go/ndn/2020")
	assert.NotNil(t, name)
	assert.NoError(t, err)
	d := ndn.NewData(name, []byte{0x01, 0x02, 0x03, 0x04})
	m := ndn.NewMetaInfo()
	m.SetContentType(0x04)
	m.SetFreshnessPeriod(time.Duration(5000) * time.Millisecond)
	m.SetFinalBlockID(ndn.NewGenericNameComponent([]byte("ndn")))
	d.SetMetaInfo(m)

	wire := []byte{tlv.Data, 0x34,
		tlv.Name, 0x0f, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.GenericNameComponent, 0x04, 0x32, 0x30, 0x32, 0x30,
		tlv.MetaInfo, 0x1b,
		tlv.ContentType, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
		tlv.FreshnessPeriod, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e,
		tlv.Content, 0x04, 0x01, 0x02, 0x03, 0x04}

	assert.False(t, d.HasWire())
	encodedBlock, err := d.Encode()
	assert.NotNil(t, encodedBlock)
	assert.NoError(t, err)
	assert.True(t, d.HasWire())

	encodedWire, err := encodedBlock.Wire()
	assert.NotNil(t, encodedWire)
	assert.NoError(t, err)
	assert.ElementsMatch(t, wire, encodedWire)
}

func TestDataWire(t *testing.T) {
	d := ndn.NewData(ndn.NewName(), []byte{})
	assert.NotNil(t, d)
	assert.False(t, d.HasWire())
	d.Encode()
	assert.True(t, d.HasWire())

	d.SetName(ndn.NewName())
	assert.False(t, d.HasWire())
	d.Encode()
	assert.True(t, d.HasWire())

	d.SetMetaInfo(ndn.NewMetaInfo())
	assert.False(t, d.HasWire())
	d.Encode()
	assert.True(t, d.HasWire())

	d.SetContent([]byte{})
	assert.False(t, d.HasWire())
	d.Encode()
	assert.True(t, d.HasWire())
}
