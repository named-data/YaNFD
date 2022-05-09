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
