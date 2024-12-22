package gen_map_test

import (
	"testing"

	enc "github.com/pulsejet/ndnd/std/encoding"
	def "github.com/pulsejet/ndnd/std/encoding/tests/gen_map"
	"github.com/pulsejet/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestStringMap(t *testing.T) {
	utils.SetTestingT(t)

	f := def.StringMap{
		Params: map[string][]byte{
			"a":  []byte("a"),
			"b":  []byte("bb"),
			"cc": []byte("cccc"),
		},
	}
	buf := f.Bytes()
	// Note: orders are not preserved, so we shouldn't check the result
	// require.Equal(t, []byte{0x85, 0x1, 0x61, 0x87, 0x1, 0x61,
	// 	0x85, 0x1, 0x62, 0x87, 0x2, 0x62, 0x62,
	// 	0x85, 0x2, 0x63, 0x63, 0x87, 0x4, 0x63, 0x63, 0x63, 0x63}, buf)
	f2 := utils.WithoutErr(def.ParseStringMap(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)

	f = def.StringMap{
		Params: map[string][]byte{},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{}, buf)
	f2 = utils.WithoutErr(def.ParseStringMap(enc.NewBufferReader(buf), false))
	require.Equal(t, 0, len(f2.Params))
}

func TestIntStructMap(t *testing.T) {
	utils.SetTestingT(t)

	f := def.IntStructMap{
		Params: map[uint64]*def.Inner{
			1: {1},
			2: {2},
			3: {3},
		},
	}
	buf := f.Bytes()
	// Note: orders are not preserved, so we shouldn't check the result
	// require.Equal(t, []byte{0x85, 0x1, 0x2, 0x87, 0x3, 0x1, 0x1, 0x2,
	// 	0x85, 0x1, 0x3, 0x87, 0x3, 0x1, 0x1, 0x3,
	// 	0x85, 0x1, 0x1, 0x87, 0x3, 0x1, 0x1, 0x1}, buf)
	f2 := utils.WithoutErr(def.ParseIntStructMap(enc.NewBufferReader(buf), false))
	require.Equal(t, f, *f2)

	f = def.IntStructMap{
		Params: map[uint64]*def.Inner{},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{}, buf)
	f2 = utils.WithoutErr(def.ParseIntStructMap(enc.NewBufferReader(buf), false))
	require.Equal(t, 0, len(f2.Params))
}
