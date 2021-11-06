/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/security"
	"github.com/named-data/YaNFD/ndn/tlv"
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
		tlv.MetaInfo, 0x0e,
		tlv.ContentType, 0x01, 0x04,
		tlv.FreshnessPeriod, 0x02, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e,
		tlv.Content, 0x04, 0x01, 0x02, 0x03, 0x04,
		tlv.SignatureInfo, 0x03, tlv.SignatureType, 0x01, 0x00,
		tlv.SignatureValue, 0x20, 0xbd, 0x04, 0x7c, 0x5e, 0xa5, 0xea, 0xca, 0x16, 0x40, 0x5d, 0x56, 0xff, 0x38, 0x32, 0xf9, 0x95, 0x4c, 0x16, 0xc5, 0xf1, 0x61, 0xac, 0x3c, 0x5a, 0x34, 0x4b, 0x59, 0x99, 0x62, 0xd8, 0x89, 0xb7})
	d, err := ndn.DecodeData(b, true)
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
	assert.NotNil(t, d.SignatureInfo())
	assert.Equal(t, security.DigestSha256Type, d.SignatureInfo().Type())
	assert.True(t, d.HasWire())
}

func TestDataDecodeCertificate(t *testing.T) {
	// https://named-data.net/ndnsec/ndn-testbed-root.ndncert.x3.base64
	cert, _ := base64.StdEncoding.DecodeString("Bv0BSQckCANuZG4IA0tFWQgI7PFMjlEjFeAIA25kbggJ/QAAAXXmfzIQFAkYAQIZBAA27oAVWzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABBsft2OBb2KNXknCL4A++JUIUHczeM6tNtXaKfLe5BnxKXxnSn9hxqZ5+P6qBfYidclGRP+zWvM8zuMU+kaSDNEWcBsBAxwWBxQIA25kbggDS0VZCAjs8UyOUSMV4P0A/Sb9AP4PMjAyMDExMjBUMTYzMTM3/QD/DzIwMjQxMjMxVDIzNTk1Of0BAif9AgAj/QIBCGZ1bGxuYW1l/QICE05ETiBUZXN0YmVkIFJvb3QgWDMXRzBFAiEA/Ia7U+qGL01yLaX8uDSINwKweLdnUIYCnIXms6goCtoCIFPAsXZhQXYOZZa6HkBxLZz2tqh3DqiLkZoY4lDYCcWp")
	b, _, _ := tlv.DecodeBlock(cert)
	d, err := ndn.DecodeData(b, false)
	assert.NotNil(t, d)
	assert.NoError(t, err)
	assert.True(t, d.HasWire())

	name, _ := ndn.NameFromString("/ndn/KEY/%EC%F1L%8EQ%23%15%E0/ndn/%FD%00%00%01u%E6%7F2%10")
	assert.True(t, d.Name().Equals(name))
	if meta := d.MetaInfo(); assert.NotNil(t, meta) {
		if ct := meta.ContentType(); assert.NotNil(t, ct) {
			assert.Equal(t, uint64(0x02), *ct)
		}
		if fp := meta.FreshnessPeriod(); assert.NotNil(t, fp) {
			assert.Equal(t, 3600*time.Second, *fp)
		}
		assert.Nil(t, meta.FinalBlockID())
	}
	assert.Len(t, d.Content(), 91)
	if si := d.SignatureInfo(); assert.NotNil(t, si) {
		assert.Equal(t, security.SignatureSha256WithEcdsaType, si.Type())
	}
	assert.Len(t, d.SignatureValue(), 71)
}

func TestDataDecodeNoSigValidation(t *testing.T) {
	b := tlv.NewBlock(tlv.Data, []byte{
		tlv.Name, 0x0f, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.GenericNameComponent, 0x04, 0x32, 0x30, 0x32, 0x30,
		tlv.MetaInfo, 0x0e,
		tlv.ContentType, 0x01, 0x04,
		tlv.FreshnessPeriod, 0x02, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e,
		tlv.Content, 0x04, 0x01, 0x02, 0x03, 0x04,
		tlv.SignatureInfo, 0x03, tlv.SignatureType, 0x01, 0x00,
		tlv.SignatureValue, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	d, err := ndn.DecodeData(b, false)
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
	assert.NotNil(t, d.SignatureInfo())
	assert.Equal(t, security.DigestSha256Type, d.SignatureInfo().Type())
	assert.True(t, d.HasWire())
}

func TestDataDecodeNullSignature(t *testing.T) {
	b := tlv.NewBlock(tlv.Data, []byte{
		tlv.Name, 0x03, tlv.GenericNameComponent, 0x01, 0x41,
		tlv.SignatureInfo, 0x03, tlv.SignatureType, 0x01, 0xc8,
		tlv.SignatureValue, 0x00})
	d, err := ndn.DecodeData(b, true)
	assert.NotNil(t, d)
	assert.NoError(t, err)
	assert.True(t, d.HasWire())

	assert.Equal(t, "/A", d.Name().String())
	assert.Equal(t, security.SignatureType(200), d.SignatureInfo().Type())
	assert.NotNil(t, d.SignatureValue())
	assert.Len(t, d.SignatureValue(), 0)
}

func TestDataDecodeUnsupportedSigType(t *testing.T) {
	b := tlv.NewBlock(tlv.Data, []byte{
		tlv.Name, 0x0f, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.GenericNameComponent, 0x04, 0x32, 0x30, 0x32, 0x30,
		tlv.MetaInfo, 0x0e,
		tlv.ContentType, 0x01, 0x04,
		tlv.FreshnessPeriod, 0x02, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e,
		tlv.Content, 0x04, 0x01, 0x02, 0x03, 0x04,
		tlv.SignatureInfo, 0x03, tlv.SignatureType, 0x01, 0x01,
		tlv.SignatureValue, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	d, err := ndn.DecodeData(b, true)
	assert.Nil(t, d)
	assert.Error(t, err)
}

func TestDataDecodeMissingSigValue(t *testing.T) {
	b := tlv.NewBlock(tlv.Data, []byte{
		tlv.Name, 0x03, tlv.GenericNameComponent, 0x01, 0x41,
		tlv.SignatureInfo, 0x03, tlv.SignatureType, 0x01, 0xc8})
	d, err := ndn.DecodeData(b, true)
	assert.Nil(t, d)
	assert.Error(t, err)
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
	s := ndn.NewSignatureInfo(security.DigestSha256Type)
	d.SetSignatureInfo(s)

	wire := []byte{tlv.Data, 0x4e,
		tlv.Name, 0x0f, tlv.GenericNameComponent, 0x02, 0x67, 0x6f, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e, tlv.GenericNameComponent, 0x04, 0x32, 0x30, 0x32, 0x30,
		tlv.MetaInfo, 0x0e,
		tlv.ContentType, 0x01, 0x04,
		tlv.FreshnessPeriod, 0x02, 0x13, 0x88,
		tlv.FinalBlockID, 0x05, tlv.GenericNameComponent, 0x03, 0x6e, 0x64, 0x6e,
		tlv.Content, 0x04, 0x01, 0x02, 0x03, 0x04,
		tlv.SignatureInfo, 0x03, tlv.SignatureType, 0x01, 0x00,
		tlv.SignatureValue, 0x20, 0xbd, 0x04, 0x7c, 0x5e, 0xa5, 0xea, 0xca, 0x16, 0x40, 0x5d, 0x56, 0xff, 0x38, 0x32, 0xf9, 0x95, 0x4c, 0x16, 0xc5, 0xf1, 0x61, 0xac, 0x3c, 0x5a, 0x34, 0x4b, 0x59, 0x99, 0x62, 0xd8, 0x89, 0xb7}

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

func TestDataEncodeUnsupportedSigType(t *testing.T) {
	name, err := ndn.NameFromString("/go/ndn/2020")
	assert.NotNil(t, name)
	assert.NoError(t, err)
	d := ndn.NewData(name, []byte{0x01, 0x02, 0x03, 0x04})
	m := ndn.NewMetaInfo()
	m.SetContentType(0x04)
	m.SetFreshnessPeriod(time.Duration(5000) * time.Millisecond)
	m.SetFinalBlockID(ndn.NewGenericNameComponent([]byte("ndn")))
	d.SetMetaInfo(m)
	s := ndn.NewSignatureInfo(security.SignatureSha256WithRsaType)
	d.SetSignatureInfo(s)

	assert.False(t, d.HasWire())
	encodedBlock, err := d.Encode()
	assert.Nil(t, encodedBlock)
	assert.Error(t, err)
	assert.False(t, d.HasWire())
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

	d.SetSignatureInfo(ndn.NewSignatureInfo(security.DigestSha256Type))
	assert.False(t, d.HasWire())
	d.Encode()
	assert.True(t, d.HasWire())
}

func BenchmarkEncode(b *testing.B) {
	name, _ := ndn.NameFromString("/go/ndn/2020")
	d := ndn.NewData(name, []byte{0x01, 0x02, 0x03, 0x04})
	m := ndn.NewMetaInfo()
	m.SetContentType(0x04)
	m.SetFreshnessPeriod(time.Duration(5000) * time.Millisecond)
	m.SetFinalBlockID(ndn.NewGenericNameComponent([]byte("ndn")))
	d.SetMetaInfo(m)
	s := ndn.NewSignatureInfo(security.DigestSha256Type)
	d.SetSignatureInfo(s)

	for i := 0; i < b.N; i++ {
		// Reset wire
		d.SetName(name)
		d.SetMetaInfo(m)
		encoded, _ := d.Encode()
		encoded.Wire()
	}
}
