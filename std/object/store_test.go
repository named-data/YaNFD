package object_test

import (
	"os"
	"testing"

	enc "github.com/pulsejet/ndnd/std/encoding"
	"github.com/pulsejet/ndnd/std/ndn"
	"github.com/pulsejet/ndnd/std/object"
	"github.com/pulsejet/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func testStoreBasic(t *testing.T, store ndn.Store) {
	name1, _ := enc.NameFromStr("/ndn/edu/ucla/test/packet/v1")
	name2, _ := enc.NameFromStr("/ndn/edu/ucla/test/packet/v5")
	name3, _ := enc.NameFromStr("/ndn/edu/ucla/test/packet/v9")
	name4, _ := enc.NameFromStr("/ndn/edu/ucla/test/packet/v2")
	name5, _ := enc.NameFromStr("/ndn/edu/arizona/test/packet/v11")

	wire1 := []byte{0x01, 0x02, 0x03}
	wire2 := []byte{0x04, 0x05, 0x06}
	wire3 := []byte{0x07, 0x08, 0x09}
	wire4 := []byte{0x0a, 0x0b, 0x0c}
	wire5 := []byte{0x0d, 0x0e, 0x0f}

	// test get when empty
	data, err := store.Get(name1, false)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	data, err = store.Get(name1, true)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	// put data
	require.NoError(t, store.Put(name1, 1, wire1))

	// exact match with full name
	data, err = store.Get(name1, false)
	require.NoError(t, err)
	require.Equal(t, wire1, data)

	// prefix match with full name
	data, err = store.Get(name1, true)
	require.NoError(t, err)
	require.Equal(t, wire1, data)

	// exact match with partial name
	name1pfx := name1[:len(name1)-1]
	data, err = store.Get(name1pfx, false)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	// prefix match with partial name
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, wire1, data)

	// insert second data under the same prefix
	require.NoError(t, store.Put(name2, 5, wire2))

	// get data2 with exact match
	data, err = store.Get(name2, false)
	require.NoError(t, err)
	require.Equal(t, wire2, data)

	// get data2 with prefix match (newer version)
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, wire2, data)

	// put data3 under the same prefix (newest)
	require.NoError(t, store.Put(name3, 9, wire3))
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, wire3, data)

	// make sure we can still get data 1
	data, err = store.Get(name1, false)
	require.NoError(t, err)
	require.Equal(t, wire1, data)

	// put data4 under the same prefix
	require.NoError(t, store.Put(name4, 2, wire4))

	// check prefix still returns data 3
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, wire3, data)

	// put data5 under a different prefix
	require.NoError(t, store.Put(name5, 2, wire5))
	data, err = store.Get(name5, false)
	require.NoError(t, err)
	require.Equal(t, wire5, data)

	// check prefix still returns data 3
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, wire3, data)

	// remove data 3
	require.NoError(t, store.Remove(name3, false))

	// check prefix now returns data 2
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, wire2, data)

	// clear subtree of name1
	require.NoError(t, store.Remove(name1pfx, true))

	// check prefix now returns no data
	data, err = store.Get(name1pfx, true)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	// check broad prefix returns data 5
	data, err = store.Get(name1[:2], true)
	require.NoError(t, err)
	require.Equal(t, wire5, data)
}

func testStoreTxn(t *testing.T, store ndn.Store) {
	txname1, _ := enc.NameFromStr("/ndn/edu/memphis/test/packet/v1")
	txname2, _ := enc.NameFromStr("/ndn/edu/memphis/test/packet/v5")
	txname3, _ := enc.NameFromStr("/ndn/edu/memphis/test/packet/v9")

	wire1 := []byte{0x01, 0x02, 0x03}
	wire2 := []byte{0x04, 0x05, 0x06}
	wire3 := []byte{0x07, 0x08, 0x09}

	// put data1 and data2 under transaction
	// verify that neither can be seen
	err := store.Begin()
	require.NoError(t, err)
	require.NoError(t, store.Put(txname1, 1, wire1))
	data, err := store.Get(txname1, false)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	require.NoError(t, store.Put(txname2, 5, wire2))
	data, err = store.Get(txname2, false)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	// commit transaction
	require.NoError(t, store.Commit())

	// verify that both data can be seen
	data, err = store.Get(txname1, false)
	require.NoError(t, err)
	require.Equal(t, wire1, data)
	data, err = store.Get(txname2, false)
	require.NoError(t, err)
	require.Equal(t, wire2, data)

	// add data3 under transaction and rollback
	err = store.Begin()
	require.NoError(t, err)
	require.NoError(t, store.Put(txname3, 9, wire3))
	data, err = store.Get(txname3, false)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)
	store.Rollback()
	data, err = store.Get(txname3, false)
	require.NoError(t, err)
	require.Equal(t, []byte(nil), data)

	// insert data3 now without transaction
	require.NoError(t, store.Put(txname3, 9, wire3))
	data, err = store.Get(txname3, false)
	require.NoError(t, err)
	require.Equal(t, wire3, data)
}

func TestMemoryStore(t *testing.T) {
	utils.SetTestingT(t)
	store := object.NewMemoryStore()
	testStoreBasic(t, store)
	testStoreTxn(t, store)
}

func TestBoltStore(t *testing.T) {
	utils.SetTestingT(t)
	filename := "test.db"
	os.Remove(filename)
	defer os.Remove(filename)

	store, err := object.NewBoltStore(filename)
	require.NoError(t, err)
	testStoreBasic(t, store)
	testStoreTxn(t, store)
	require.NoError(t, store.Close())
}
