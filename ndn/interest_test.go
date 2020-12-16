/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/stretchr/testify/assert"
)

func TestInterestCreate(t *testing.T) {
	i := ndn.NewInterest(ndn.NewName().Append(ndn.NewGenericNameComponent([]byte("go"))).Append(ndn.NewGenericNameComponent([]byte("ndn"))))
	assert.Equal(t, 4, len(i.Nonce()))

	interestString := "Interest(Name=/go/ndn, Nonce=0x" + hex.EncodeToString(i.Nonce()) + ", Lifetime=4000ms)"
	assert.Equal(t, interestString, i.String())

	assert.Equal(t, false, i.CanBePrefix())
	assert.Equal(t, false, i.MustBeFresh())
	assert.Equal(t, 0, len(i.ForwardingHint()))
	assert.Equal(t, 4000*time.Millisecond, i.Lifetime())
	assert.Nil(t, i.HopLimit())
	assert.Equal(t, 0, len(i.ApplicationParameters()))
}

func TestInterestDecode(t *testing.T) {
	block := tlv.NewBlock(tlv.Interest,
		[]byte{
			tlv.Name, 0x2B, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.ParametersSha256DigestComponent, 0x20, 0x09, 0x01, 0xA2, 0xD0, 0x4B, 0xB8, 0x8A, 0xB8, 0x19, 0x13, 0xC2, 0x32, 0xA3, 0xEF, 0xC8, 0x9F, 0xAC, 0xF8, 0xB3, 0x2D, 0xF2, 0x0E, 0x3D, 0x43, 0x53, 0x89, 0xF5, 0x50, 0x27, 0x25, 0xC0, 0x4F,
			tlv.CanBePrefix, 0x00,
			tlv.MustBeFresh, 0x00,
			tlv.ForwardingHint, 0x14, tlv.Delegation, 0x12, tlv.Preference, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A, tlv.Name, 0x06, tlv.GenericNameComponent, 0x04, 0x75, 0x63, 0x6c, 0x61,
			tlv.Nonce, 0x04, 0x01, 0x02, 0x03, 0x04,
			tlv.InterestLifetime, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe8,
			tlv.HopLimit, 0x01, 0x40,
			tlv.ApplicationParameters, 0x00,
			0xAA, 0x04, 0xBB, 0xCC, 0xDD, 0xEE,
			0xBB, 0x06, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66})

	i, err := ndn.DecodeInterest(block)
	assert.NotNil(t, i)
	assert.NoError(t, err)
	assert.Equal(t, "/go/ndn/params-sha256=0901a2d04bb88ab81913c232a3efc89facf8b32df20e3d435389f5502725c04f", i.Name().String())
	assert.Equal(t, true, i.CanBePrefix())
	assert.Equal(t, true, i.MustBeFresh())
	assert.Equal(t, 1, len(i.ForwardingHint()))
	assert.Equal(t, uint64(0x0a), i.ForwardingHint()[0].Preference())
	assert.Equal(t, "/ucla", i.ForwardingHint()[0].Name().String())
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, i.Nonce())
	assert.Equal(t, 1000*time.Millisecond, i.Lifetime())
	assert.Equal(t, uint8(0x40), *i.HopLimit())
	assert.Equal(t, 3, len(i.ApplicationParameters()))
	assert.Equal(t, "Interest(Name=/go/ndn/params-sha256=0901a2d04bb88ab81913c232a3efc89facf8b32df20e3d435389f5502725c04f, CanBePrefix, MustBeFresh, ForwardingHint(Delegation(10, /ucla)), Nonce=0x01020304, Lifetime=1000ms, HopLimit=64, ApplicationParameters)", i.String())
}

func TestInterestEncode(t *testing.T) {
	rawBlock := tlv.NewBlock(tlv.Interest,
		[]byte{
			tlv.Name, 0x2B, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.ParametersSha256DigestComponent, 0x20, 0x09, 0x01, 0xA2, 0xD0, 0x4B, 0xB8, 0x8A, 0xB8, 0x19, 0x13, 0xC2, 0x32, 0xA3, 0xEF, 0xC8, 0x9F, 0xAC, 0xF8, 0xB3, 0x2D, 0xF2, 0x0E, 0x3D, 0x43, 0x53, 0x89, 0xF5, 0x50, 0x27, 0x25, 0xC0, 0x4F,
			tlv.CanBePrefix, 0x00,
			tlv.MustBeFresh, 0x00,
			tlv.ForwardingHint, 0x14, tlv.Delegation, 0x12, tlv.Preference, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A, tlv.Name, 0x06, tlv.GenericNameComponent, 0x04, 0x75, 0x63, 0x6c, 0x61,
			tlv.Nonce, 0x04, 0x01, 0x02, 0x03, 0x04,
			tlv.InterestLifetime, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe8,
			tlv.HopLimit, 0x01, 0x40,
			tlv.ApplicationParameters, 0x00,
			0xAA, 0x04, 0xBB, 0xCC, 0xDD, 0xEE,
			0xBB, 0x06, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66})
	rawBlockWire, err := rawBlock.Wire()
	assert.NotNil(t, rawBlockWire)
	assert.NoError(t, err)

	i, err := ndn.DecodeInterest(rawBlock)
	assert.NotNil(t, i)
	assert.NoError(t, err)

	i.SetCanBePrefix(false)
	assert.False(t, i.HasWire())
	i.SetCanBePrefix(true)
	assert.False(t, i.HasWire())
	encodedBlock, err := i.Encode()
	assert.NotNil(t, encodedBlock)
	assert.NoError(t, err)
	encodedWire, err := encodedBlock.Wire()
	assert.NoError(t, err)
	assert.ElementsMatch(t, rawBlockWire, encodedWire)
	assert.True(t, i.HasWire())
}

func TestForwardingHint(t *testing.T) {
	i := ndn.NewInterest(ndn.NewName().Append(ndn.NewGenericNameComponent([]byte("go"))).Append(ndn.NewGenericNameComponent([]byte("ndn"))))
	assert.Equal(t, 0, len(i.ForwardingHint()))

	name1, err := ndn.NameFromString("/ucla")
	assert.NotNil(t, name1)
	assert.NoError(t, err)
	d1, err := ndn.NewDelegation(10, name1)
	assert.NotNil(t, d1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(i.ForwardingHint()))
	i.AppendForwardingHint(d1)
	assert.Equal(t, 1, len(i.ForwardingHint()))
	assert.Equal(t, uint64(10), i.ForwardingHint()[0].Preference())
	assert.Equal(t, "/ucla", i.ForwardingHint()[0].Name().String())

	name2, err := ndn.NameFromString("/arizona")
	assert.NotNil(t, name2)
	assert.NoError(t, err)
	d2, err := ndn.NewDelegation(20, name2)
	assert.NotNil(t, d2)
	assert.NoError(t, err)
	i.AppendForwardingHint(d2)
	assert.Equal(t, 2, len(i.ForwardingHint()))
	assert.Equal(t, uint64(10), i.ForwardingHint()[0].Preference())
	assert.Equal(t, "/ucla", i.ForwardingHint()[0].Name().String())
	assert.Equal(t, uint64(20), i.ForwardingHint()[1].Preference())
	assert.Equal(t, "/arizona", i.ForwardingHint()[1].Name().String())

	assert.NoError(t, i.EraseForwardingHint(1))
	assert.Equal(t, 1, len(i.ForwardingHint()))
	assert.Equal(t, uint64(10), i.ForwardingHint()[0].Preference())
	assert.Equal(t, "/ucla", i.ForwardingHint()[0].Name().String())

	i.ClearForwardingHints()
	assert.Equal(t, 0, len(i.ForwardingHint()))
}

func TestApplicationParameters(t *testing.T) {
	name, err := ndn.NameFromString("/go/ndn/seg=100")
	assert.NotNil(t, name)
	assert.NoError(t, err)
	i := ndn.NewInterest(name)

	app1 := tlv.NewBlock(0xAA, []byte{0x11, 0x22, 0x33, 0x44})
	assert.Equal(t, 0, len(i.ApplicationParameters()))
	i.AppendApplicationParameter(app1)
	assert.Equal(t, 2, len(i.ApplicationParameters()))
	assert.Equal(t, uint32(tlv.ApplicationParameters), i.ApplicationParameters()[0].Type())
	assert.Equal(t, uint32(0xAA), i.ApplicationParameters()[1].Type())

	i.ClearApplicationParameters()
	assert.Equal(t, 0, len(i.ApplicationParameters()))
	app2 := tlv.NewBlock(tlv.ApplicationParameters, []byte{0x11, 0x22, 0x33, 0x44})
	i.AppendApplicationParameter(app2)
	assert.Equal(t, 1, len(i.ApplicationParameters()))
	assert.Equal(t, uint32(tlv.ApplicationParameters), i.ApplicationParameters()[0].Type())
	i.AppendApplicationParameter(app1)
	assert.Equal(t, 2, len(i.ApplicationParameters()))
	assert.Equal(t, uint32(tlv.ApplicationParameters), i.ApplicationParameters()[0].Type())
	assert.Equal(t, uint32(0xAA), i.ApplicationParameters()[1].Type())
}
