package dummy_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/engine/dummy"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

func TestBasicConsume(t *testing.T) {
	utils.SetTestingT(t)

	testOnData := func(r enc.ParseReader) error {
		t.Fatal("No data should be received in this test.")
		return nil
	}
	// onError is not actually called by dummy face.
	testOnError := func(err error) error {
		require.NoError(t, err)
		return err
	}

	face := dummy.NewDummyFace()
	utils.WithErr(face.Consume())
	require.Error(t, face.Open())
	face.SetCallback(testOnData, testOnError)
	require.NoError(t, face.Open())
	utils.WithErr(face.Consume())

	err := face.Send(enc.Wire{enc.Buffer{0x05, 0x03, 0x01, 0x02, 0x03}})
	require.NoError(t, err)
	data := utils.WithoutErr(face.Consume())
	require.Equal(t, enc.Buffer{0x05, 0x03, 0x01, 0x02, 0x03}, data)
	utils.WithErr(face.Consume())

	err = face.Send(enc.Wire{enc.Buffer{0x05, 0x01, 0x01}})
	require.NoError(t, err)
	err = face.Send(enc.Wire{enc.Buffer{0x05, 0x04, 0x01, 0x02, 0x03, 0x04}})
	require.NoError(t, err)
	data = utils.WithoutErr(face.Consume())
	require.Equal(t, enc.Buffer{0x05, 0x01, 0x01}, data)
	data = utils.WithoutErr(face.Consume())
	require.Equal(t, enc.Buffer{0x05, 0x04, 0x01, 0x02, 0x03, 0x04}, data)
	utils.WithErr(face.Consume())

	require.NoError(t, face.Close())
}

func TestBasicFeed(t *testing.T) {
	utils.SetTestingT(t)
	cnt := 0

	testOnData := func(r enc.ParseReader) error {
		cnt++
		switch cnt {
		case 1:
			buf := utils.WithoutErr(r.ReadBuf(r.Length()))
			require.Equal(t, enc.Buffer{0x05, 0x03, 0x01, 0x02, 0x03}, buf)
			return nil
		case 2:
			buf := utils.WithoutErr(r.ReadBuf(r.Length()))
			require.Equal(t, enc.Buffer{0x05, 0x01, 0x01}, buf)
			return nil
		case 3:
			buf := utils.WithoutErr(r.ReadBuf(r.Length()))
			require.Equal(t, enc.Buffer{0x05, 0x04, 0x01, 0x02, 0x03, 0x04}, buf)
			return nil
		}
		t.Fatal("No data should be received now.")
		return nil
	}
	testOnError := func(err error) error {
		require.NoError(t, err)
		return err
	}

	face := dummy.NewDummyFace()
	face.SetCallback(testOnData, testOnError)
	require.NoError(t, face.Open())

	err := face.FeedPacket(enc.Buffer{0x05, 0x03, 0x01, 0x02, 0x03})
	require.NoError(t, err)
	err = face.FeedPacket(enc.Buffer{0x05, 0x01, 0x01})
	require.NoError(t, err)
	err = face.FeedPacket(enc.Buffer{0x05, 0x04, 0x01, 0x02, 0x03, 0x04})
	require.NoError(t, err)

	require.Equal(t, 3, cnt)
	require.NoError(t, face.Close())
}
