package table

import (
	"bytes"
	"sort"
	"testing"
	"time"

	"github.com/named-data/YaNFD/ndn"
	"github.com/stretchr/testify/assert"
)

func TestNewPitCSTree(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS()

	// Initialization size should be 0
	assert.Equal(t, pitCS.PitSize(), 0)
	assert.Equal(t, pitCS.CsSize(), 0)

	// Any search should return nil
	// Interest is some random string
	name, _ := ndn.NameFromString("/interest1")
	interest := ndn.NewInterest(name)
	pitEntry := pitCS.FindInterestExactMatch(interest)
	assert.Nil(t, pitEntry)
	data := ndn.NewData(name, []byte("abc"))
	pitEntries := pitCS.FindInterestPrefixMatchByData(data, nil)
	assert.Equal(t, len(pitEntries), 0)

	// Test searching CS
	csEntry := pitCS.FindMatchingDataFromCS(interest)
	assert.Nil(t, csEntry)

	// Interest is root - still should not match
	name, _ = ndn.NameFromString("/")
	interest = ndn.NewInterest(name)
	pitEntry = pitCS.FindInterestExactMatch(interest)
	assert.Nil(t, pitEntry)
	data = ndn.NewData(name, []byte("abc"))
	pitEntries = pitCS.FindInterestPrefixMatchByData(data, nil)
	assert.Equal(t, len(pitEntries), 0)

	csEntry = pitCS.FindMatchingDataFromCS(interest)
	assert.Nil(t, csEntry)
}

func TestIsCsAdmitting(t *testing.T) {
	csAdmit = false
	csReplacementPolicy = "lru"

	pitCS := NewPitCS()
	assert.Equal(t, pitCS.IsCsAdmitting(), csAdmit)

	csAdmit = true
	pitCS = NewPitCS()
	assert.Equal(t, pitCS.IsCsAdmitting(), csAdmit)
}

func TestIsCsServing(t *testing.T) {
	csServe = false
	csReplacementPolicy = "lru"

	pitCS := NewPitCS()
	assert.Equal(t, pitCS.IsCsServing(), csServe)

	csServe = true
	pitCS = NewPitCS()
	assert.Equal(t, pitCS.IsCsServing(), csServe)
}

func TestInsertInterest(t *testing.T) {
	// Interest does not already exist
	hint, _ := ndn.NameFromString("/")
	inFace := uint64(1111)
	name, _ := ndn.NameFromString("/interest1")
	interest := ndn.NewInterest(name)

	csReplacementPolicy = "lru"

	pitCS := NewPitCS()

	pitEntry, duplicateNonce := pitCS.InsertInterest(interest, hint, inFace)

	assert.False(t, duplicateNonce)

	assert.True(t, pitEntry.Name().Equals(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.CanBePrefix())
	assert.Equal(t, pitEntry.MustBeFresh(), interest.MustBeFresh())
	assert.True(t, pitEntry.ForwardingHint().Equals(hint))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 1)

	// Interest already exists, so we should just update it
	// insert the interest again, the same data should be returned
	pitEntry, duplicateNonce = pitCS.InsertInterest(interest, hint, inFace)

	assert.False(t, duplicateNonce)

	assert.True(t, pitEntry.Name().Equals(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.CanBePrefix())
	assert.Equal(t, pitEntry.MustBeFresh(), interest.MustBeFresh())
	assert.True(t, pitEntry.ForwardingHint().Equals(hint))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 1)

	// Looping interest, duplicate nonce
	pitEntry.InsertInRecord(interest, inFace, []byte("abc"))
	inFace2 := uint64(2222)
	pitEntry, duplicateNonce = pitCS.InsertInterest(interest, hint, inFace2)

	assert.True(t, duplicateNonce)
	assert.True(t, pitEntry.Name().Equals(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.CanBePrefix())
	assert.Equal(t, pitEntry.MustBeFresh(), interest.MustBeFresh())
	assert.True(t, pitEntry.ForwardingHint().Equals(hint))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 1)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 1)

	// Insert another distinct interest to check it works
	hint2, _ := ndn.NameFromString("/")
	inFace3 := uint64(3333)
	name2, _ := ndn.NameFromString("/interest2")
	interest2 := ndn.NewInterest(name2)
	pitEntry, duplicateNonce = pitCS.InsertInterest(interest2, hint2, inFace3)

	assert.False(t, duplicateNonce)
	assert.True(t, pitEntry.Name().Equals(name2))
	assert.Equal(t, pitEntry.CanBePrefix(), interest2.CanBePrefix())
	assert.Equal(t, pitEntry.MustBeFresh(), interest2.MustBeFresh())
	assert.True(t, pitEntry.ForwardingHint().Equals(hint2))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 2)

	// PitCS with 2 interests, prefixes of each other.
	pitCS = NewPitCS()

	hint, _ = ndn.NameFromString("/")
	inFace = uint64(4444)
	name, _ = ndn.NameFromString("/interest")
	interest = ndn.NewInterest(name)

	name2, _ = ndn.NameFromString("/interest/longer")
	interest2 = ndn.NewInterest(name2)
	pitEntry, duplicateNonce = pitCS.InsertInterest(interest, hint, inFace)
	pitEntry2, duplicateNonce2 := pitCS.InsertInterest(interest2, hint, inFace)

	assert.False(t, duplicateNonce)
	assert.True(t, pitEntry.Name().Equals(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.CanBePrefix())
	assert.Equal(t, pitEntry.MustBeFresh(), interest.MustBeFresh())
	assert.True(t, pitEntry.ForwardingHint().Equals(hint))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.False(t, duplicateNonce2)
	assert.True(t, pitEntry2.Name().Equals(name2))
	assert.Equal(t, pitEntry2.CanBePrefix(), interest2.CanBePrefix())
	assert.Equal(t, pitEntry2.MustBeFresh(), interest2.MustBeFresh())
	assert.True(t, pitEntry2.ForwardingHint().Equals(hint))
	assert.False(t, pitEntry2.Satisfied())

	assert.Equal(t, len(pitEntry2.InRecords()), 0)
	assert.Equal(t, len(pitEntry2.OutRecords()), 0)
	assert.Equal(t, pitEntry2.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry2.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 2)
}

func TestRemoveInterest(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS()
	hint, _ := ndn.NameFromString("/")
	inFace := uint64(1111)
	name1, _ := ndn.NameFromString("/interest1")
	interest1 := ndn.NewInterest(name1)

	// Simple insert and removal
	pitEntry, _ := pitCS.InsertInterest(interest1, hint, inFace)
	removedInterest := pitCS.RemoveInterest(pitEntry)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 0)

	// Remove a nonexistent pit entry
	name2, _ := ndn.NameFromString("/interest2")
	interest2 := ndn.NewInterest(name2)
	pitEntry2, _ := pitCS.InsertInterest(interest2, hint, inFace)

	removedInterest = pitCS.RemoveInterest(pitEntry)
	assert.False(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 1)
	removedInterest = pitCS.RemoveInterest(pitEntry2)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 0)

	// Remove a pit entry from a node with more than 1 pit entry
	hint2, _ := ndn.NameFromString("/2")
	hint3, _ := ndn.NameFromString("/3")
	hint4, _ := ndn.NameFromString("/4")
	_, _ = pitCS.InsertInterest(interest2, hint, inFace)
	_, _ = pitCS.InsertInterest(interest2, hint2, inFace)
	pitEntry3, _ := pitCS.InsertInterest(interest2, hint3, inFace)
	_, _ = pitCS.InsertInterest(interest2, hint4, inFace)

	removedInterest = pitCS.RemoveInterest(pitEntry3)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 3)

	// Remove PIT entry from a node with more than 1 child
	pitCS = NewPitCS()
	name1, _ = ndn.NameFromString("/root/1")
	name2, _ = ndn.NameFromString("/root/2")
	name3, _ := ndn.NameFromString("/root/3")
	interest1 = ndn.NewInterest(name1)
	interest2 = ndn.NewInterest(name2)
	interest3 := ndn.NewInterest(name3)

	_, _ = pitCS.InsertInterest(interest1, hint, inFace)
	pitEntry2, _ = pitCS.InsertInterest(interest2, hint, inFace)
	_, _ = pitCS.InsertInterest(interest3, hint3, inFace)

	removedInterest = pitCS.RemoveInterest(pitEntry2)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 2)
}

func TestFindInterestExactMatch(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS()
	hint, _ := ndn.NameFromString("/")
	inFace := uint64(1111)
	name, _ := ndn.NameFromString("/interest1")
	interest := ndn.NewInterest(name)

	// Simple insert and find
	_, _ = pitCS.InsertInterest(interest, hint, inFace)

	pitEntry := pitCS.FindInterestExactMatch(interest)
	assert.NotNil(t, pitEntry)
	assert.True(t, pitEntry.Name().Equals(interest.Name()))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.CanBePrefix())
	assert.Equal(t, pitEntry.MustBeFresh(), interest.MustBeFresh())
	assert.True(t, pitEntry.ForwardingHint().Equals(hint))
	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.False(t, pitEntry.Satisfied())

	// Look for nonexistent name
	name2, _ := ndn.NameFromString("/nonexistent")
	interest2 := ndn.NewInterest(name2)
	pitEntryNil := pitCS.FindInterestExactMatch(interest2)
	assert.Nil(t, pitEntryNil)

	// /a exists but we're looking for /a/b
	longername, _ := ndn.NameFromString("/interest1/more_name_content")
	interest3 := ndn.NewInterest(longername)

	pitEntryNil = pitCS.FindInterestExactMatch(interest3)
	assert.Nil(t, pitEntryNil)

	// /a/b exists but we're looking for /a only
	pitCS.RemoveInterest(pitEntry)
	_, _ = pitCS.InsertInterest(interest3, hint, inFace)
	pitEntryNil = pitCS.FindInterestExactMatch(interest)
	assert.Nil(t, pitEntryNil)
}

func TestFindInterestPrefixMatchByData(t *testing.T) {
	// Basically the same as FindInterestPrefixMatch, but with data instead
	csReplacementPolicy = "lru"
	pitCS := NewPitCS()
	name, _ := ndn.NameFromString("/interest1")
	data := ndn.NewData(name, []byte("abc"))
	hint, _ := ndn.NameFromString("/")
	inFace := uint64(1111)
	interest := ndn.NewInterest(name)
	interest.SetCanBePrefix(true)

	// Simple insert and find
	_, _ = pitCS.InsertInterest(interest, hint, inFace)

	pitEntries := pitCS.FindInterestPrefixMatchByData(data, nil)
	assert.Equal(t, len(pitEntries), 1)
	assert.True(t, pitEntries[0].Name().Equals(interest.Name()))
	assert.Equal(t, pitEntries[0].CanBePrefix(), interest.CanBePrefix())
	assert.Equal(t, pitEntries[0].MustBeFresh(), interest.MustBeFresh())
	assert.True(t, pitEntries[0].ForwardingHint().Equals(hint))
	assert.Equal(t, len(pitEntries[0].InRecords()), 0)
	assert.Equal(t, len(pitEntries[0].OutRecords()), 0)
	assert.False(t, pitEntries[0].Satisfied())

	// Look for nonexistent name
	name2, _ := ndn.NameFromString("/nonexistent")
	data2 := ndn.NewData(name2, []byte("abc"))
	pitEntriesEmpty := pitCS.FindInterestPrefixMatchByData(data2, nil)
	assert.Equal(t, len(pitEntriesEmpty), 0)

	// /a exists but we're looking for /a/b, return just /a
	longername, _ := ndn.NameFromString("/interest1/more_name_content")
	interest3 := ndn.NewInterest(longername)
	data3 := ndn.NewData(longername, []byte("abc"))

	pitEntriesEmpty = pitCS.FindInterestPrefixMatchByData(data3, nil)
	assert.Equal(t, len(pitEntriesEmpty), 1)

	// /a/b exists but we're looking for /a
	// should return both /a/b and /a
	_, _ = pitCS.InsertInterest(interest3, hint, inFace)
	pitEntries = pitCS.FindInterestPrefixMatchByData(data3, nil)
	assert.Equal(t, len(pitEntries), 2)
}

func TestInsertOutRecord(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS()
	name, _ := ndn.NameFromString("/interest1")
	hint, _ := ndn.NameFromString("/")
	inFace := uint64(1111)
	interest := ndn.NewInterest(name)
	interest.SetCanBePrefix(true)

	// New outrecord
	pitEntry, _ := pitCS.InsertInterest(interest, hint, inFace)
	outRecord := pitEntry.InsertOutRecord(interest, inFace)
	assert.Equal(t, outRecord.Face, inFace)
	assert.Equal(t, outRecord.LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecord.LatestNonce, interest.Nonce()))

	// Update existing outrecord
	var oldNonce []byte
	copy(interest.Nonce(), oldNonce) // Need to copy by value, or else we save a reference
	interest.SetNonce([]byte("nonce"))
	outRecord = pitEntry.InsertOutRecord(interest, inFace)
	assert.Equal(t, outRecord.Face, inFace)
	assert.Equal(t, outRecord.LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecord.LatestNonce, interest.Nonce()))
	assert.False(t, bytes.Equal(outRecord.LatestNonce, oldNonce))

	// Add new outrecord on a different face
	inFace2 := uint64(2222)
	outRecord = pitEntry.InsertOutRecord(interest, inFace2)
	assert.Equal(t, outRecord.Face, inFace2)
	assert.Equal(t, outRecord.LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecord.LatestNonce, interest.Nonce()))
}

func TestGetOutRecords(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS()
	name, _ := ndn.NameFromString("/interest1")
	hint, _ := ndn.NameFromString("/")
	inFace := uint64(1111)
	interest := ndn.NewInterest(name)
	interest.SetCanBePrefix(true)

	// New outrecord
	pitEntry, _ := pitCS.InsertInterest(interest, hint, inFace)
	_ = pitEntry.InsertOutRecord(interest, inFace)
	outRecords := pitEntry.GetOutRecords()
	assert.Equal(t, len(outRecords), 1)
	assert.Equal(t, outRecords[0].Face, inFace)
	assert.Equal(t, outRecords[0].LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecords[0].LatestNonce, interest.Nonce()))

	// Update existing outrecord
	var oldNonce []byte
	copy(interest.Nonce(), oldNonce) // Need to copy by value, or else we save a reference
	interest.SetNonce([]byte("nonce"))
	_ = pitEntry.InsertOutRecord(interest, inFace)
	outRecords = pitEntry.GetOutRecords()
	assert.Equal(t, len(outRecords), 1)
	assert.Equal(t, outRecords[0].Face, inFace)
	assert.Equal(t, outRecords[0].LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecords[0].LatestNonce, interest.Nonce()))
	assert.False(t, bytes.Equal(outRecords[0].LatestNonce, oldNonce))

	// Add new outrecord on a different face
	inFace2 := uint64(2222)
	_ = pitEntry.InsertOutRecord(interest, inFace2)
	outRecords = pitEntry.GetOutRecords()
	sort.Slice(outRecords, func(i, j int) bool {
		// Sort by face ID
		return outRecords[i].Face < outRecords[j].Face
	})
	assert.Equal(t, len(outRecords), 2)

	assert.Equal(t, outRecords[0].Face, inFace)
	assert.Equal(t, outRecords[0].LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecords[0].LatestNonce, interest.Nonce()))

	assert.Equal(t, outRecords[1].Face, inFace2)
	assert.Equal(t, outRecords[1].LatestInterest, interest)
	assert.True(t, bytes.Equal(outRecords[1].LatestNonce, interest.Nonce()))
}

func TestInsertData(t *testing.T) {
	csReplacementPolicy = "lru"
	csCapacity = 1024
	pitCS := NewPitCS()

	// Data does not already exist
	name1, _ := ndn.NameFromString("/interest1")
	interest1 := ndn.NewInterest(name1)
	interest1.SetCanBePrefix(false)
	data1 := ndn.NewData(name1, []byte("data1"))

	pitCS.InsertData(data1)
	csEntry1 := pitCS.FindMatchingDataFromCS(interest1)
	assert.Equal(t, pitCS.CsSize(), 1)
	assert.True(t, bytes.Equal(csEntry1.Data().Content(), data1.Content()))

	// Insert data associated with same name, so we should just update it
	// Should not result in a new CsEntry
	pitCS.InsertData(data1)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.Equal(t, pitCS.CsSize(), 1)
	assert.True(t, bytes.Equal(csEntry1.Data().Content(), data1.Content()))

	// Insert some different data, should result in creation of new CsEntry
	name2, _ := ndn.NameFromString("/interest2")
	interest2 := ndn.NewInterest(name2)
	interest2.SetCanBePrefix(false)
	data2 := ndn.NewData(name2, []byte("data2"))
	pitCS.InsertData(data2)

	csEntry2 := pitCS.FindMatchingDataFromCS(interest2)
	assert.Equal(t, pitCS.CsSize(), 2)
	assert.True(t, bytes.Equal(csEntry2.Data().Content(), data2.Content()))

	// PitCS with interest /a/b, data /a/b/v=10
	// Interest /a/b is prefix allowed
	// Should return data with name /a/b/v=10
	pitCS = NewPitCS()

	name1, _ = ndn.NameFromString("/a/b")
	interest1 = ndn.NewInterest(name1)
	interest1.SetCanBePrefix(true)
	name2, _ = ndn.NameFromString("/a/b/v=10")
	data2 = ndn.NewData(name2, []byte("data2"))

	pitCS.InsertData(data2)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.Equal(t, pitCS.CsSize(), 1)
	assert.True(t, bytes.Equal(csEntry1.Data().Content(), data2.Content()))

	// Reduced CS capacity to check that eviction occurs
	csCapacity = 1
	pitCS = NewPitCS()
	data1 = ndn.NewData(name1, []byte("data1"))
	data2 = ndn.NewData(name2, []byte("data2"))
	pitCS.InsertData(data1)
	pitCS.InsertData(data2)
	assert.Equal(t, pitCS.CsSize(), 1)
}

func FindMatchingDataFromCS(t *testing.T) {
	csReplacementPolicy = "lru"
	csCapacity = 1024
	pitCS := NewPitCS()

	// Insert data and then fetch it
	name1, _ := ndn.NameFromString("/interest1")
	interest1 := ndn.NewInterest(name1)
	interest1.SetCanBePrefix(false)
	data1 := ndn.NewData(name1, []byte("data1"))

	pitCS.InsertData(data1)
	csEntry1 := pitCS.FindMatchingDataFromCS(interest1)
	assert.True(t, bytes.Equal(csEntry1.Data().Content(), data1.Content()))

	// Insert some different data
	name2, _ := ndn.NameFromString("/interest2")
	interest2 := ndn.NewInterest(name2)
	interest2.SetCanBePrefix(false)
	data2 := ndn.NewData(name2, []byte("data2"))
	pitCS.InsertData(data2)

	csEntry2 := pitCS.FindMatchingDataFromCS(interest2)
	assert.True(t, bytes.Equal(csEntry2.Data().Content(), data2.Content()))

	// Look for nonexistent data
	nameNonExistent, _ := ndn.NameFromString("/doesnotexist")
	interestNonExistent := ndn.NewInterest(nameNonExistent)
	csEntryNonExistent := pitCS.FindMatchingDataFromCS(interestNonExistent)
	assert.Nil(t, csEntryNonExistent)

	// PitCS with interest /a/b, data /a/b/v=10
	// Interest /a/b is prefix allowed
	// Should return data with name /a/b/v=10
	pitCS = NewPitCS()

	name1, _ = ndn.NameFromString("/a/b")
	interest1 = ndn.NewInterest(name1)
	interest1.SetCanBePrefix(true)
	name2, _ = ndn.NameFromString("/a/b/v=10")
	data2 = ndn.NewData(name2, []byte("data2"))

	pitCS.InsertData(data2)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.True(t, bytes.Equal(csEntry1.Data().Content(), data2.Content()))

	// PitCS with interest /a/b
	// Now look for interest /a/b with prefix allowed
	// Should return nil since there is no data
	pitCS = NewPitCS()

	name1, _ = ndn.NameFromString("/a/b")
	interest1 = ndn.NewInterest(name1)
	interest1.SetCanBePrefix(true)

	pitCS.InsertInterest(interest1, nil, 0)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.Nil(t, csEntry1)
}
