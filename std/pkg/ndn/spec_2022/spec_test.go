package spec_2022_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	"github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

func TestMakeDataBasic(t *testing.T) {
	utils.SetTestingT(t)

	spec := spec_2022.Spec{}

	wire, _, err := spec.MakeData(
		utils.WithoutErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
		},
		nil,
		security.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06\x42\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x14\x03\x18\x01\x00"+
			"\x16\x03\x1b\x01\x00"+
			"\x17 \x7f1\xe4\t\xc5z/\x1d\r\xdaVh8\xfd\xd9\x94"+
			"\xd8'S\x13[\xd7\x15\xa5\x9d%^\x80\xf2\xab\xf0\xb5"),
		wire.Join())

	wire, _, err = spec.MakeData(
		utils.WithoutErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
		},
		enc.Wire{[]byte("01020304")},
		security.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06L\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x14\x03\x18\x01\x00"+
			"\x15\x0801020304"+
			"\x16\x03\x1b\x01\x00"+
			"\x17 \x94\xe9\xda\x91\x1a\x11\xfft\x02i:G\x0cO\xdd!"+
			"\xe0\xc7\xb6\xfd\x8f\x9cn\xc5\x93{\x93\x04\xe0\xdf\xa6S"),
		wire.Join())

	wire, _, err = spec.MakeData(
		utils.WithoutErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
		},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06\x1b\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x14\x03\x18\x01\x00"),
		wire.Join())

	wire, _, err = spec.MakeData(
		utils.WithoutErr(enc.NameFromStr("/E")),
		&ndn.DataConfig{
			ContentType: nil,
		},
		enc.Wire{},
		security.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, utils.WithoutErr(hex.DecodeString(
		"06300703080145"+
			"1400150016031b0100"+
			"1720f965ee682c6973c3cbaa7b69e4c7063680f83be93a46be2ccc98686134354b66")),
		wire.Join())
}

func TestMakeDataMetaInfo(t *testing.T) {
	utils.SetTestingT(t)
	spec := spec_2022.Spec{}

	wire, _, err := spec.MakeData(
		utils.WithoutErr(enc.NameFromStr("/local/ndn/prefix/37=%00")),
		&ndn.DataConfig{
			ContentType:  utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:    utils.IdPtr(1000 * time.Millisecond),
			FinalBlockID: utils.IdPtr(enc.NewSequenceNumComponent(2)),
		},
		nil,
		security.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06\x4e\x07\x17\x08\x05local\x08\x03ndn\x08\x06prefix\x25\x01\x00"+
			"\x14\x0c\x18\x01\x00\x19\x02\x03\xe8\x1a\x03\x3a\x01\x02"+
			"\x16\x03\x1b\x01\x00"+
			"\x17 \x0f^\xa1\x0c\xa7\xf5Fb\xf0\x9cOT\xe0FeC\x8f92\x04\x9d\xabP\x80o'\x94\xaa={hQ"),
		wire.Join())
}

type testSigner struct{}

func (_ testSigner) SigInfo(ndn.Data) (*ndn.SigConfig, error) {
	name, _ := enc.NameFromStr("/KEY")
	return &ndn.SigConfig{
		Type:    ndn.SigType(200),
		KeyName: name,
	}, nil
}

func (_ testSigner) EstimateSize() uint {
	return 10
}

func (_ testSigner) ComputeSigValue(enc.Wire) ([]byte, error) {
	return []byte{0, 0, 0, 0, 0}, nil
}

func TestMakeDataShrink(t *testing.T) {
	utils.SetTestingT(t)
	spec := spec_2022.Spec{}

	wire, _, err := spec.MakeData(
		utils.WithoutErr(enc.NameFromStr("/test")),
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
		},
		nil,
		testSigner{},
	)
	require.NoError(t, err)
	require.Equal(t, []byte{
		0x6, 0x22, 0x7, 0x6, 0x8, 0x4, 0x74, 0x65, 0x73, 0x74, 0x14, 0x3, 0x18, 0x1, 0x0,
		0x16, 0xc, 0x1b, 0x1, 0xc8, 0x1c, 0x7, 0x7, 0x5, 0x8, 0x3, 0x4b, 0x45, 0x59,
		0x17, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0},
		wire.Join())
}

func TestReadDataBasic(t *testing.T) {
	utils.SetTestingT(t)
	spec := spec_2022.Spec{}

	data, covered, err := spec.ReadData(enc.NewBufferReader([]byte(
		"\x06\x42\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x14\x03\x18\x01\x00" +
			"\x16\x03\x1b\x01\x00" +
			"\x17 \x7f1\xe4\t\xc5z/\x1d\r\xdaVh8\xfd\xd9\x94" +
			"\xd8'S\x13[\xd7\x15\xa5\x9d%^\x80\xf2\xab\xf0\xb5"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, *data.ContentType())
	require.True(t, data.Freshness() == nil)
	require.True(t, data.FinalBlockID() == nil)
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.True(t, data.Content() == nil)
	h := sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig := h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())

	data, covered, err = spec.ReadData(enc.NewBufferReader([]byte(
		"\x06L\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x14\x03\x18\x01\x00" +
			"\x15\x0801020304" +
			"\x16\x03\x1b\x01\x00" +
			"\x17 \x94\xe9\xda\x91\x1a\x11\xfft\x02i:G\x0cO\xdd!" +
			"\xe0\xc7\xb6\xfd\x8f\x9cn\xc5\x93{\x93\x04\xe0\xdf\xa6S"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, *data.ContentType())
	require.True(t, data.Freshness() == nil)
	require.True(t, data.FinalBlockID() == nil)
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.Equal(t, []byte("01020304"), data.Content().Join())
	h = sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig = h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())

	data, _, err = spec.ReadData(enc.NewBufferReader([]byte(
		"\x06\x1b\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x14\x03\x18\x01\x00"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, *data.ContentType())
	require.True(t, data.Freshness() == nil)
	require.True(t, data.FinalBlockID() == nil)
	require.Equal(t, ndn.SignatureNone, data.Signature().SigType())
	require.True(t, data.Content() == nil)
	require.True(t, data.Signature().SigValue() == nil)

	data, covered, err = spec.ReadData(enc.NewBufferReader(utils.WithoutErr(hex.DecodeString(
		"06300703080145" +
			"1400150016031b0100" +
			"1720f965ee682c6973c3cbaa7b69e4c7063680f83be93a46be2ccc98686134354b66"),
	)))
	require.NoError(t, err)
	require.Equal(t, "/E", data.Name().String())
	require.True(t, data.ContentType() == nil)
	require.True(t, data.Freshness() == nil)
	require.True(t, data.FinalBlockID() == nil)
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.Equal(t, 0, len(data.Content().Join()))
	h = sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig = h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())
}

func TestReadDataMetaInfo(t *testing.T) {
	utils.SetTestingT(t)
	spec := spec_2022.Spec{}

	data, covered, err := spec.ReadData(enc.NewBufferReader([]byte(
		"\x06\x4e\x07\x17\x08\x05local\x08\x03ndn\x08\x06prefix\x25\x01\x00" +
			"\x14\x0c\x18\x01\x00\x19\x02\x03\xe8\x1a\x03\x3a\x01\x02" +
			"\x16\x03\x1b\x01\x00" +
			"\x17 \x0f^\xa1\x0c\xa7\xf5Fb\xf0\x9cOT\xe0FeC\x8f92\x04\x9d\xabP\x80o'\x94\xaa={hQ"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix/37=%00", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, *data.ContentType())
	require.Equal(t, 1000*time.Millisecond, *data.Freshness())
	require.Equal(t, enc.NewSequenceNumComponent(2), *data.FinalBlockID())
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.True(t, data.Content() == nil)
	h := sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig := h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())
}
