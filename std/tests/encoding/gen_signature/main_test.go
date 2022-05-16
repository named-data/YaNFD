package gen_signature_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/utils"
	def "github.com/zjkmxy/go-ndn/tests/encoding/gen_signature"
)

func TestT1(t *testing.T) {
	utils.SetTestingT(t)

	f := &def.T1{
		H1: 1,
		H2: utils.ConstPtr[uint64](2),
		C: enc.Wire{
			[]byte{0x01, 0x02, 0x03},
			[]byte{0x04, 0x05, 0x06},
		},
	}
	wire, cov := f.Encode(5, []byte{0x07, 0x08, 0x09})
	require.GreaterOrEqual(t, len(wire), 5)
	require.GreaterOrEqual(t, len(cov), 1)
	require.Equal(t, []byte{
		0x01, 0x01, 0x01, 0x02, 0x01, 0x02,
		0x03, 0x06, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
		0x04, 0x03, 0x07, 0x08, 0x09,
	}, wire.Join())
	require.Equal(t, []byte{
		0x02, 0x01, 0x02,
		0x03, 0x06, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
	}, cov.Join())
	f2, cov2, err := def.ReadT1(enc.NewWireReader(wire))
	require.NoError(t, err)
	require.Equal(t, f.H1, f2.H1)
	require.Equal(t, f.H2, f2.H2)
	require.Equal(t, f.C.Join(), f2.C.Join())
	require.Equal(t, []byte{0x07, 0x08, 0x09}, f2.Sig.Join())
	require.Equal(t, cov.Join(), cov2.Join())

	wire, _ = f.Encode(0, nil)
	require.GreaterOrEqual(t, len(wire), 3)
	require.Equal(t, []byte{
		0x01, 0x01, 0x01, 0x02, 0x01, 0x02,
		0x03, 0x06, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
	}, wire.Join())
	f2, _, err = def.ReadT1(enc.NewWireReader(wire))
	require.NoError(t, err)
	require.Equal(t, f.H1, f2.H1)
	require.Equal(t, f.H2, f2.H2)
	require.Equal(t, f.C.Join(), f2.C.Join())
	require.Equal(t, 0, len(f2.Sig.Join()))

	f = &def.T1{
		H1: 4,
		H2: nil,
		C: enc.Wire{
			[]byte{0x01, 0x02, 0x03},
		},
	}
	wire, cov = f.Encode(1, []byte{0x01})
	require.GreaterOrEqual(t, len(wire), 4)
	require.GreaterOrEqual(t, len(cov), 1)
	require.Equal(t, []byte{
		0x01, 0x01, 0x04,
		0x03, 0x03, 0x01, 0x02, 0x03,
		0x04, 0x01, 0x01,
	}, wire.Join())
	require.Equal(t, []byte{0x03, 0x03, 0x01, 0x02, 0x03}, cov.Join())
	f2, cov2, err = def.ReadT1(enc.NewWireReader(wire))
	require.NoError(t, err)
	require.Equal(t, f.H1, f2.H1)
	require.True(t, f2.H2 == nil)
	require.Equal(t, f.C.Join(), f2.C.Join())
	require.Equal(t, []byte{0x01}, f2.Sig.Join())
	require.Equal(t, cov.Join(), cov2.Join())

	f = &def.T1{
		H1: 0,
		H2: nil,
		C:  enc.Wire{},
	}
	wire, cov = f.Encode(1, []byte{0x01})
	require.GreaterOrEqual(t, len(wire), 2)
	require.GreaterOrEqual(t, len(cov), 1)
	require.Equal(t, []byte{
		0x01, 0x01, 0x00,
		0x03, 0x00,
		0x04, 0x01, 0x01,
	}, wire.Join())
	require.Equal(t, []byte{0x03, 0x00}, cov.Join())
	f2, cov2, err = def.ReadT1(enc.NewWireReader(wire))
	require.NoError(t, err)
	require.Equal(t, f.H1, f2.H1)
	require.True(t, f2.H2 == nil)
	require.Equal(t, enc.Wire{}, f2.C)
	require.Equal(t, []byte{0x01}, f2.Sig.Join())
	require.Equal(t, cov.Join(), cov2.Join())

	f = &def.T1{
		H1: 0,
		H2: nil,
		C:  nil,
	}
	wire, cov = f.Encode(1, []byte{0x01})
	require.GreaterOrEqual(t, len(wire), 2)
	require.GreaterOrEqual(t, len(cov), 1)
	require.Equal(t, []byte{
		0x01, 0x01, 0x00,
		0x04, 0x01, 0x01,
	}, wire.Join())
	require.Equal(t, 0, len(cov.Join()))
	f2, cov2, err = def.ReadT1(enc.NewWireReader(wire))
	require.NoError(t, err)
	require.Equal(t, f.H1, f2.H1)
	require.True(t, f2.H2 == nil)
	require.True(t, f2.C == nil)
	require.Equal(t, []byte{0x01}, f2.Sig.Join())
	require.Equal(t, 0, len(cov2.Join()))
}
