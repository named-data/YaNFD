package table

import (
	"bytes"
	"testing"
	"time"

	"github.com/named-data/YaNFD/ndn"

	"github.com/stretchr/testify/assert"
)

func TestBasePitEntryGetters(t *testing.T) {
	name, _ := ndn.NameFromString("/something")
	currTime := time.Now()
	bpe := basePitEntry{
		name:           name,
		canBePrefix:    true,
		mustBeFresh:    true,
		forwardingHint: name,
		expirationTime: currTime,
		satisfied:      true,
		token:          1234,
	}

	assert.True(t, bpe.Name().Equals(name))
	assert.Equal(t, bpe.CanBePrefix(), true)
	assert.Equal(t, bpe.MustBeFresh(), true)
	assert.True(t, bpe.ForwardingHint().Equals(name))
	assert.Equal(t, len(bpe.InRecords()), 0)
	assert.Equal(t, len(bpe.OutRecords()), 0)
	assert.Equal(t, bpe.ExpirationTime(), currTime)
	assert.Equal(t, bpe.Satisfied(), true)
	assert.Equal(t, bpe.Token(), uint32(1234))
}

func TestBasePitEntrySetters(t *testing.T) {
	name, _ := ndn.NameFromString("/something")
	currTime := time.Now()
	bpe := basePitEntry{
		name:           name,
		canBePrefix:    true,
		mustBeFresh:    true,
		forwardingHint: name,
		expirationTime: currTime,
		satisfied:      true,
		token:          1234,
	}

	newTime := time.Now()
	bpe.SetExpirationTime(newTime)
	assert.Equal(t, bpe.ExpirationTime(), newTime)

	bpe.SetSatisfied(false)
	assert.Equal(t, bpe.Satisfied(), false)
}

func TestClearInRecords(t *testing.T) {
	inrecord1 := PitInRecord{}
	inrecord2 := PitInRecord{}
	inRecords := map[uint64]*PitInRecord{
		1: &inrecord1,
		2: &inrecord2,
	}
	bpe := basePitEntry{
		inRecords: inRecords,
	}
	assert.NotEqual(t, len(bpe.InRecords()), 0)
	bpe.ClearInRecords()
	assert.Equal(t, len(bpe.InRecords()), 0)
}

func TestClearOutRecords(t *testing.T) {
	outrecord1 := PitOutRecord{}
	outrecord2 := PitOutRecord{}
	outRecords := map[uint64]*PitOutRecord{
		1: &outrecord1,
		2: &outrecord2,
	}
	bpe := basePitEntry{
		outRecords: outRecords,
	}
	assert.NotEqual(t, len(bpe.OutRecords()), 0)
	bpe.ClearOutRecords()
	assert.Equal(t, len(bpe.OutRecords()), 0)
}

func TestInsertInRecord(t *testing.T) {
	// Case 1: interest does not already exist in basePitEntry.inRecords
	name, _ := ndn.NameFromString("/something")
	interest := ndn.NewInterest(name)
	pitToken := []byte("abc")
	bpe := basePitEntry{
		inRecords: make(map[uint64]*PitInRecord),
	}
	faceID := uint64(1234)
	inRecord, alreadyExists := bpe.InsertInRecord(interest, faceID, pitToken)
	assert.False(t, alreadyExists)
	assert.Equal(t, inRecord.Face, faceID)
	assert.Equal(t, bytes.Compare(inRecord.LatestNonce, interest.Nonce()), 0)
	assert.Equal(t, inRecord.LatestInterest, interest)
	assert.Equal(t, bytes.Compare(inRecord.PitToken, pitToken), 0)
	assert.Equal(t, len(bpe.InRecords()), 1)

	record, ok := bpe.InRecords()[faceID]
	assert.True(t, ok)
	assert.Equal(t, record, inRecord)

	// Case 2: interest already exists in basePitEntry.inRecords
	interest.ResetNonce() // get a "new" interest by resetting its nonce
	inRecord, alreadyExists = bpe.InsertInRecord(interest, faceID, pitToken)
	assert.True(t, alreadyExists)
	assert.Equal(t, inRecord.Face, faceID)
	assert.Equal(t, bytes.Compare(inRecord.LatestNonce, interest.Nonce()), 0)
	assert.Equal(t, inRecord.LatestInterest, interest)
	assert.Equal(t, bytes.Compare(inRecord.PitToken, pitToken), 0)
	assert.Equal(t, len(bpe.InRecords()), 1) // should update the original record in place
	record, ok = bpe.InRecords()[faceID]
	assert.True(t, ok)
	assert.Equal(t, record, inRecord)

	// Add another inRecord
	name2, _ := ndn.NameFromString("/another_something")
	interest2 := ndn.NewInterest(name2)
	pitToken2 := []byte("xyz")
	faceID2 := uint64(6789)
	inRecord, alreadyExists = bpe.InsertInRecord(interest2, faceID2, pitToken2)
	assert.False(t, alreadyExists)
	assert.Equal(t, inRecord.Face, faceID2)
	assert.Equal(t, bytes.Compare(inRecord.LatestNonce, interest2.Nonce()), 0)
	assert.Equal(t, inRecord.LatestInterest, interest2)
	assert.Equal(t, bytes.Compare(inRecord.PitToken, pitToken2), 0)
	assert.Equal(t, len(bpe.InRecords()), 2) // should be a new inRecord
	record, ok = bpe.InRecords()[faceID2]
	assert.True(t, ok)
	assert.Equal(t, record, inRecord)

	// TODO: For unit testing the timestamps and expiration times, the time
	// module needs to be mocked so that we can control the return value
	// of time.Now()
}

func TestBaseCsEntryGetters(t *testing.T) {
	name, _ := ndn.NameFromString("/something")
	currTime := time.Now()
	data := ndn.NewData(name, []byte("abc"))
	bpe := baseCsEntry{
		index:     1234,
		staleTime: currTime,
		data:      data,
	}

	assert.Equal(t, bpe.Index(), uint64(1234))
	assert.Equal(t, bpe.StaleTime(), currTime)
	assert.Equal(t, bpe.Data(), data)
}
