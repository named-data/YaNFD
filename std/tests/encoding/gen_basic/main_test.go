package gen_basic_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/utils"
	"github.com/zjkmxy/go-ndn/tests/encoding/gen_basic"
)

func TestFakeMetaInfo(t *testing.T) {
	utils.SetTestingT(t)

	f := gen_basic.FakeMetaInfo{
		Number: 1,
		Time:   2 * time.Second,
		Binary: []byte{3, 4, 5},
	}
	buf := f.Bytes()
	require.Equal(t,
		[]byte{
			0x18, 0x01, 0x01,
			0x19, 0x02, 0x07, 0xd0,
			0x1a, 0x03, 0x03, 0x04, 0x05,
		},
		buf)

	f2 := utils.WithoutErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)

	buf2 := []byte{
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
	}
	utils.WithErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferReader(buf2), false))

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x08, 0x03, 0x04, 0x05,
	}
	utils.WithErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferReader(buf2), false))

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
		0x30, 0x01, 0x00,
	}
	f2 = utils.WithoutErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferReader(buf2), false))
	require.Equal(t, f, *f2)

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
		0x31, 0x01, 0x00,
	}
	f2 = utils.WithoutErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferReader(buf2), true))
	require.Equal(t, f, *f2)

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
		0x31, 0x01, 0x00,
	}
	utils.WithErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferReader(buf2), false))
}

func TestOptField(t *testing.T) {
	utils.SetTestingT(t)

	f := gen_basic.OptField{
		Number: utils.ConstPtr[uint64](1),
		Time:   utils.ConstPtr(2 * time.Second),
		Binary: []byte{3, 4, 5},
	}
	buf := f.Bytes()
	require.Equal(t,
		[]byte{
			0x18, 0x01, 0x01,
			0x19, 0x02, 0x07, 0xd0,
			0x1a, 0x03, 0x03, 0x04, 0x05,
		},
		buf)
	f2 := utils.WithoutErr(gen_basic.ParseOptField(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)

	f = gen_basic.OptField{
		Number: nil,
		Time:   nil,
		Binary: nil,
	}
	buf = f.Bytes()
	require.Equal(t,
		[]byte{},
		buf)
	f2 = utils.WithoutErr(gen_basic.ParseOptField(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)

	f = gen_basic.OptField{
		Number: utils.ConstPtr[uint64](0),
		Time:   utils.ConstPtr(0 * time.Second),
		Binary: []byte{},
	}
	buf = f.Bytes()
	require.Equal(t,
		[]byte{
			0x18, 0x01, 0x00,
			0x19, 0x01, 0x00,
			0x1a, 0x00,
		},
		buf)
	f2 = utils.WithoutErr(gen_basic.ParseOptField(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)
}
