package gen_composition_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/utils"
	def "github.com/zjkmxy/go-ndn/tests/encoding/gen_composition"
)

func TestIntArray(t *testing.T) {
	utils.SetTestingT(t)

	f := def.IntArray{
		Words: []uint64{1, 2, 3},
	}
	buf := f.Bytes()
	require.Equal(t, []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x02, 0x01, 0x01, 0x03}, buf)
	f2 := utils.WithoutErr(def.ParseIntArray(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)

	f = def.IntArray{
		Words: []uint64{},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{}, buf)
	f2 = utils.WithoutErr(def.ParseIntArray(enc.NewBufferReader(buf), false))
	require.Equal(t, 0, len(f2.Words))
}

func TestNameArray(t *testing.T) {
	utils.SetTestingT(t)

	f := def.NameArray{
		Names: []enc.Name{
			utils.WithoutErr(enc.NameFromStr("/A/B")),
			utils.WithoutErr(enc.NameFromStr("/C")),
		},
	}
	buf := f.Bytes()
	require.Equal(t, []byte{
		0x07, 0x06, 0x08, 0x01, 'A', 0x08, 0x01, 'B',
		0x07, 0x03, 0x08, 0x01, 'C'}, buf)
	f2 := utils.WithoutErr(def.ParseNameArray(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)
}

func TestNested(t *testing.T) {
	utils.SetTestingT(t)

	f := def.Nested{
		Val: &def.Inner{
			Num: 255,
		},
	}
	buf := f.Bytes()
	require.Equal(t, []byte{0x02, 0x03, 0x01, 0x01, 0xff}, buf)
	f2 := utils.WithoutErr(def.ParseNested(enc.NewBufferReader(buf), false))
	require.Equal(t, f.Val.Num, f2.Val.Num)

	f = def.Nested{
		Val: nil,
	}
	buf = f.Bytes()
	require.Equal(t, 0, len(buf))
	f2 = utils.WithoutErr(def.ParseNested(enc.NewBufferReader(buf), false))
	require.True(t, f2.Val == nil)
}

func TestNestedSeq(t *testing.T) {
	utils.SetTestingT(t)

	f := def.NestedSeq{
		Vals: []*def.Inner{
			{Num: 255},
			{Num: 256},
		},
	}
	buf := f.Bytes()
	require.Equal(t, []byte{
		0x03, 0x03, 0x01, 0x01, 0xff,
		0x03, 0x04, 0x01, 0x02, 0x01, 0x00,
	}, buf)
	f2 := utils.WithoutErr(def.ParseNestedSeq(enc.NewBufferReader(buf), false))
	require.Equal(t, 2, len(f2.Vals))
	require.Equal(t, uint64(255), f2.Vals[0].Num)
	require.Equal(t, uint64(256), f2.Vals[1].Num)

	f = def.NestedSeq{
		Vals: nil,
	}
	buf = f.Bytes()
	require.Equal(t, 0, len(buf))
	f2 = utils.WithoutErr(def.ParseNestedSeq(enc.NewBufferReader(buf), false))
	require.Equal(t, 0, len(f2.Vals))
}

func TestNestedWire(t *testing.T) {
	utils.SetTestingT(t)

	f := def.NestedWire{
		W1: &def.InnerWire1{
			Wire1: enc.Wire{
				[]byte{1, 2, 3},
				[]byte{4, 5, 6},
			},
			Num: utils.IdPtr[uint64](255),
		},
		N: 13,
		W2: &def.InnerWire2{
			Wire2: enc.Wire{
				[]byte{7, 8, 9},
				[]byte{10, 11, 12},
			},
		},
	}
	wire := f.Encode()
	require.GreaterOrEqual(t, len(wire), 6)
	buf := wire.Join()
	require.Equal(t, []byte{
		0x04, 0x0b,
		0x01, 0x06, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x02, 0x01, 0xff,
		0x05, 0x01, 0x0d,
		0x06, 0x08,
		0x03, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
	}, buf)
	f2 := utils.WithoutErr(def.ParseNestedWire(enc.NewWireReader(wire), false))
	require.Equal(t, f.W1.Wire1.Join(), f2.W1.Wire1.Join())
	require.Equal(t, f.W1.Num, f2.W1.Num)
	require.Equal(t, f.N, f2.N)
	require.Equal(t, f.W2.Wire2.Join(), f2.W2.Wire2.Join())

	f = def.NestedWire{
		W1: &def.InnerWire1{
			Wire1: enc.Wire{},
			Num:   nil,
		},
		N: 0,
		W2: &def.InnerWire2{
			Wire2: enc.Wire{},
		},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{
		0x04, 0x02,
		0x01, 0x00,
		0x05, 0x01, 0,
		0x06, 0x02,
		0x03, 0x00,
	}, buf)
	f2 = utils.WithoutErr(def.ParseNestedWire(enc.NewBufferReader(buf), false))
	require.Equal(t, 0, len(f2.W1.Wire1.Join()))
	require.False(t, f2.W1.Wire1 == nil)
	require.Equal(t, 0, len(f2.W2.Wire2.Join()))
	require.False(t, f2.W2.Wire2 == nil)

	f = def.NestedWire{
		W1: &def.InnerWire1{
			Wire1: nil,
			Num:   nil,
		},
		N: 0,
		W2: &def.InnerWire2{
			Wire2: nil,
		},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{0x04, 0x00, 0x05, 0x01, 0, 0x06, 0x00}, buf)
	f2 = utils.WithoutErr(def.ParseNestedWire(enc.NewBufferReader(buf), false))
	require.Equal(t, enc.Wire(nil), f2.W1.Wire1)
	require.Equal(t, enc.Wire(nil), f2.W2.Wire2)

	f = def.NestedWire{
		W1: nil,
		N:  0,
		W2: nil,
	}
	buf = f.Bytes()
	require.Equal(t, []byte{0x05, 0x01, 0}, buf)
	f2 = utils.WithoutErr(def.ParseNestedWire(enc.NewBufferReader(buf), false))
	require.True(t, f2.W1 == nil)
	require.True(t, f2.W2 == nil)
}
