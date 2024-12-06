package table

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/named-data/YaNFD/ndn_defn"
	"github.com/stretchr/testify/assert"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
)

func makeData(name enc.Name, content enc.Wire) *ndn_defn.PendingPacket {
	data := &spec_2022.Data{
		NameV:    name,
		ContentV: content,
	}
	netPacket := new(ndn_defn.PendingPacket)
	netPacket.EncPacket = new(spec_2022.Packet)
	netPacket.EncPacket.Data = data
	return netPacket
}

func makeInterest(name enc.Name) *ndn_defn.PendingPacket {
	val := new(uint32)
	*val = rand.Uint32()
	interest := &spec_2022.Interest{
		NameV:  name,
		NonceV: val,
	}
	netPacket := new(ndn_defn.PendingPacket)
	netPacket.EncPacket = new(spec_2022.Packet)
	netPacket.EncPacket.Interest = interest
	return netPacket
}

func TestNewPitCSTree(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS(func(PitEntry) {})

	// Initialization size should be 0
	assert.Equal(t, pitCS.PitSize(), 0)
	assert.Equal(t, pitCS.CsSize(), 0)

	// Any search should return nil
	// Interest is some random string
	name, _ := enc.NameFromStr("/interest1")
	interest := makeInterest(name)
	pitEntry := pitCS.FindInterestExactMatchEnc(interest)
	assert.Nil(t, pitEntry)
	data := makeData(name, enc.Wire{})
	pitEntries := pitCS.FindInterestPrefixMatchByDataEnc(data, nil)
	assert.Equal(t, len(pitEntries), 0)

	// Test searching CS
	csEntry := pitCS.FindMatchingDataFromCS(interest)
	assert.Nil(t, csEntry)

	// Interest is root - still should not match
	name, _ = enc.NameFromStr("/")
	interest2 := makeInterest(name)
	pitEntry = pitCS.FindInterestExactMatchEnc(interest2)
	assert.Nil(t, pitEntry)

	data2 := makeData(name, enc.Wire{})
	pitEntries = pitCS.FindInterestPrefixMatchByDataEnc(data2, nil)
	assert.Equal(t, len(pitEntries), 0)

	csEntry = pitCS.FindMatchingDataFromCS(interest2)
	assert.Nil(t, csEntry)
}

func TestIsCsAdmitting(t *testing.T) {
	csAdmit = false
	csReplacementPolicy = "lru"

	pitCS := NewPitCS(func(PitEntry) {})
	assert.Equal(t, pitCS.IsCsAdmitting(), csAdmit)

	csAdmit = true
	pitCS = NewPitCS(func(PitEntry) {})
	assert.Equal(t, pitCS.IsCsAdmitting(), csAdmit)
}

func TestIsCsServing(t *testing.T) {
	csServe = false
	csReplacementPolicy = "lru"

	pitCS := NewPitCS(func(PitEntry) {})
	assert.Equal(t, pitCS.IsCsServing(), csServe)

	csServe = true
	pitCS = NewPitCS(func(PitEntry) {})
	assert.Equal(t, pitCS.IsCsServing(), csServe)
}

func TestInsertInterest(t *testing.T) {
	// Interest does not already exist
	hint, _ := enc.NameFromStr("/")
	inFace := uint64(1111)
	name, _ := enc.NameFromStr("/interest1")
	interest := makeInterest(name)

	csReplacementPolicy = "lru"

	pitCS := NewPitCS(func(PitEntry) {})

	pitEntry, duplicateNonce := pitCS.InsertInterest(interest, hint, inFace)

	assert.False(t, duplicateNonce)

	assert.True(t, pitEntry.EncName().Equal(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry.MustBeFresh(), interest.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry.ForwardingHintNew().Equal(hint))
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

	assert.True(t, pitEntry.EncName().Equal(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry.MustBeFresh(), interest.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry.ForwardingHintNew().Equal(hint))
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
	assert.True(t, pitEntry.EncName().Equal(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry.MustBeFresh(), interest.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry.ForwardingHintNew().Equal(hint))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 1)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 1)

	// Insert another distinct interest to check it works
	hint2, _ := enc.NameFromStr("/")
	inFace3 := uint64(3333)
	name2, _ := enc.NameFromStr("/interest2")
	interest2 := makeInterest(name2)
	pitEntry, duplicateNonce = pitCS.InsertInterest(interest2, hint2, inFace3)

	assert.False(t, duplicateNonce)
	assert.True(t, pitEntry.EncName().Equal(name2))
	assert.Equal(t, pitEntry.CanBePrefix(), interest2.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry.MustBeFresh(), interest2.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry.ForwardingHintNew().Equal(hint2))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.Equal(t, pitCS.PitSize(), 2)

	// PitCS with 2 interests, prefixes of each other.
	pitCS = NewPitCS(func(PitEntry) {})

	hint, _ = enc.NameFromStr("/")
	inFace = uint64(4444)
	name, _ = enc.NameFromStr("/interest")
	interest = makeInterest(name)

	name2, _ = enc.NameFromStr("/interest/longer")
	interest2 = makeInterest(name2)
	pitEntry, duplicateNonce = pitCS.InsertInterest(interest, hint, inFace)
	pitEntry2, duplicateNonce2 := pitCS.InsertInterest(interest2, hint, inFace)

	assert.False(t, duplicateNonce)
	// assert.True(t, pitEntry.Name().Equals(name))
	// assert.Equal(t, pitEntry.CanBePrefix(), interest.CanBePrefix())
	// assert.Equal(t, pitEntry.MustBeFresh(), interest.MustBeFresh())
	// assert.True(t, pitEntry.ForwardingHint().Equals(hint))
	assert.True(t, pitEntry.EncName().Equal(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry.MustBeFresh(), interest.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry.ForwardingHintNew().Equal(hint))
	assert.False(t, pitEntry.Satisfied())

	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.Equal(t, pitEntry.PitCs(), pitCS)
	// expiration time should be cancelled upon receiving a new interest
	assert.Equal(t, pitEntry.ExpirationTime(), time.Unix(0, 0))

	assert.False(t, duplicateNonce2)
	// assert.True(t, pitEntry2.Name().Equals(name2))
	// assert.Equal(t, pitEntry2.CanBePrefix(), interest2.CanBePrefix())
	// assert.Equal(t, pitEntry2.MustBeFresh(), interest2.MustBeFresh())
	// assert.True(t, pitEntry2.ForwardingHint().Equals(hint))
	assert.True(t, pitEntry2.EncName().Equal(name2))
	assert.Equal(t, pitEntry2.CanBePrefix(), interest2.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry2.MustBeFresh(), interest2.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry2.ForwardingHintNew().Equal(hint2))
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
	pitCS := NewPitCS(func(PitEntry) {})
	hint, _ := enc.NameFromStr("/")
	inFace := uint64(1111)
	name1, _ := enc.NameFromStr("/interest1")
	interest1 := makeInterest(name1)

	// Simple insert and removal
	pitEntry, _ := pitCS.InsertInterest(interest1, hint, inFace)
	removedInterest := pitCS.RemoveInterest(pitEntry)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 0)

	// Remove a nonexistent pit entry
	name2, _ := enc.NameFromStr("/interest2")
	interest2 := makeInterest(name2)
	pitEntry2, _ := pitCS.InsertInterest(interest2, hint, inFace)

	removedInterest = pitCS.RemoveInterest(pitEntry)
	assert.False(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 1)
	removedInterest = pitCS.RemoveInterest(pitEntry2)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 0)

	// Remove a pit entry from a node with more than 1 pit entry
	hint2, _ := enc.NameFromStr("/2")
	hint3, _ := enc.NameFromStr("/3")
	hint4, _ := enc.NameFromStr("/4")
	_, _ = pitCS.InsertInterest(interest2, hint, inFace)
	_, _ = pitCS.InsertInterest(interest2, hint2, inFace)
	pitEntry3, _ := pitCS.InsertInterest(interest2, hint3, inFace)
	_, _ = pitCS.InsertInterest(interest2, hint4, inFace)

	removedInterest = pitCS.RemoveInterest(pitEntry3)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 3)

	// Remove PIT entry from a node with more than 1 child
	pitCS = NewPitCS(func(PitEntry) {})
	name1, _ = enc.NameFromStr("/root/1")
	name2, _ = enc.NameFromStr("/root/2")
	name3, _ := enc.NameFromStr("/root/3")
	interest1 = makeInterest(name1)
	interest2 = makeInterest(name2)
	interest3 := makeInterest(name3)

	_, _ = pitCS.InsertInterest(interest1, hint, inFace)
	pitEntry2, _ = pitCS.InsertInterest(interest2, hint, inFace)
	_, _ = pitCS.InsertInterest(interest3, hint3, inFace)

	removedInterest = pitCS.RemoveInterest(pitEntry2)
	assert.True(t, removedInterest)
	assert.Equal(t, pitCS.PitSize(), 2)
}

func TestFindInterestExactMatch(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS(func(PitEntry) {})
	hint, _ := enc.NameFromStr("/")
	inFace := uint64(1111)
	name, _ := enc.NameFromStr("/interest1")
	interest := makeInterest(name)

	// Simple insert and find
	_, _ = pitCS.InsertInterest(interest, hint, inFace)

	pitEntry := pitCS.FindInterestExactMatchEnc(interest)
	assert.NotNil(t, pitEntry)
	assert.True(t, pitEntry.EncName().Equal(name))
	assert.Equal(t, pitEntry.CanBePrefix(), interest.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntry.MustBeFresh(), interest.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntry.ForwardingHintNew().Equal(hint))
	assert.Equal(t, len(pitEntry.InRecords()), 0)
	assert.Equal(t, len(pitEntry.OutRecords()), 0)
	assert.False(t, pitEntry.Satisfied())

	// Look for nonexistent name
	name2, _ := enc.NameFromStr("/nonexistent")
	interest2 := makeInterest(name2)
	pitEntryNil := pitCS.FindInterestExactMatchEnc(interest2)
	assert.Nil(t, pitEntryNil)

	// /a exists but we're looking for /a/b
	longername, _ := enc.NameFromStr("/interest1/more_name_content")
	interest3 := makeInterest(longername)

	pitEntryNil = pitCS.FindInterestExactMatchEnc(interest3)
	assert.Nil(t, pitEntryNil)

	// /a/b exists but we're looking for /a only
	pitCS.RemoveInterest(pitEntry)
	_, _ = pitCS.InsertInterest(interest3, hint, inFace)
	pitEntryNil = pitCS.FindInterestExactMatchEnc(interest)
	assert.Nil(t, pitEntryNil)
}

func TestFindInterestPrefixMatchByData(t *testing.T) {
	// Basically the same as FindInterestPrefixMatch, but with data instead
	csReplacementPolicy = "lru"
	pitCS := NewPitCS(func(PitEntry) {})
	name, _ := enc.NameFromStr("/interest1")
	data := makeData(name, enc.Wire{})
	hint, _ := enc.NameFromStr("/")
	inFace := uint64(1111)
	interest := makeInterest(name)
	interest.EncPacket.Interest.CanBePrefixV = true

	// Simple insert and find
	_, _ = pitCS.InsertInterest(interest, hint, inFace)

	pitEntries := pitCS.FindInterestPrefixMatchByDataEnc(data, nil)
	assert.Equal(t, len(pitEntries), 1)
	assert.True(t, pitEntries[0].EncName().Equal(interest.EncPacket.Interest.NameV))
	assert.Equal(t, pitEntries[0].CanBePrefix(), interest.EncPacket.Interest.CanBePrefixV)
	assert.Equal(t, pitEntries[0].MustBeFresh(), interest.EncPacket.Interest.MustBeFreshV)
	assert.True(t, pitEntries[0].ForwardingHintNew().Equal(hint))
	assert.Equal(t, len(pitEntries[0].InRecords()), 0)
	assert.Equal(t, len(pitEntries[0].OutRecords()), 0)
	assert.False(t, pitEntries[0].Satisfied())

	// Look for nonexistent name
	name2, _ := enc.NameFromStr("/nonexistent")
	data2 := makeData(name2, enc.Wire{})
	pitEntriesEmpty := pitCS.FindInterestPrefixMatchByDataEnc(data2, nil)
	assert.Equal(t, len(pitEntriesEmpty), 0)

	// /a exists but we're looking for /a/b, return just /a
	longername, _ := enc.NameFromStr("/interest1/more_name_content")
	interest3 := makeInterest(longername)
	data3 := makeData(longername, enc.Wire{})

	pitEntriesEmpty = pitCS.FindInterestPrefixMatchByDataEnc(data3, nil)
	assert.Equal(t, len(pitEntriesEmpty), 1)

	// /a/b exists but we're looking for /a
	// should return both /a/b and /a
	_, _ = pitCS.InsertInterest(interest3, hint, inFace)
	pitEntries = pitCS.FindInterestPrefixMatchByDataEnc(data3, nil)
	assert.Equal(t, len(pitEntries), 2)
}

func TestInsertOutRecord(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS(func(PitEntry) {})
	name, _ := enc.NameFromStr("/interest1")
	hint, _ := enc.NameFromStr("/")
	inFace := uint64(1111)
	interest := makeInterest(name)
	interest.EncPacket.Interest.CanBePrefixV = true

	// New outrecord
	pitEntry, _ := pitCS.InsertInterest(interest, hint, inFace)
	outRecord := pitEntry.InsertOutRecord(interest, inFace)
	assert.Equal(t, outRecord.Face, inFace)
	assert.Equal(t, outRecord.LatestEncInterest, interest)
	assert.True(t, outRecord.LatestEncNonce == *interest.EncPacket.Interest.NonceV)

	// Update existing outrecord
	oldNonce := new(uint32)
	*oldNonce = 2
	*interest.EncPacket.Interest.NonceV = *oldNonce
	*interest.EncPacket.Interest.NonceV = 3
	outRecord = pitEntry.InsertOutRecord(interest, inFace)
	assert.Equal(t, outRecord.Face, inFace)
	assert.Equal(t, outRecord.LatestEncInterest, interest)
	assert.True(t, outRecord.LatestEncNonce == *interest.EncPacket.Interest.NonceV)
	assert.False(t, outRecord.LatestEncNonce == *oldNonce)

	// Add new outrecord on a different face
	inFace2 := uint64(2222)
	outRecord = pitEntry.InsertOutRecord(interest, inFace2)
	assert.Equal(t, outRecord.Face, inFace2)
	assert.Equal(t, outRecord.LatestEncInterest, interest)
	assert.True(t, outRecord.LatestEncNonce == *interest.EncPacket.Interest.NonceV)
}

func TestGetOutRecords(t *testing.T) {
	csReplacementPolicy = "lru"
	pitCS := NewPitCS(func(PitEntry) {})
	name, _ := enc.NameFromStr("/interest1")
	hint, _ := enc.NameFromStr("/")
	inFace := uint64(1111)
	interest := makeInterest(name)
	interest.EncPacket.Interest.CanBePrefixV = true

	// New outrecord
	pitEntry, _ := pitCS.InsertInterest(interest, hint, inFace)
	_ = pitEntry.InsertOutRecord(interest, inFace)
	outRecords := pitEntry.GetOutRecords()
	assert.Equal(t, len(outRecords), 1)
	assert.Equal(t, outRecords[0].Face, inFace)
	assert.Equal(t, outRecords[0].LatestEncInterest, interest)
	assert.True(t, outRecords[0].LatestEncNonce == *interest.EncPacket.Interest.NonceV)

	// Update existing outrecord
	oldNonce := new(uint32)
	*oldNonce = 2
	*interest.EncPacket.Interest.NonceV = *oldNonce
	*interest.EncPacket.Interest.NonceV = 3
	_ = pitEntry.InsertOutRecord(interest, inFace)
	outRecords = pitEntry.GetOutRecords()
	assert.Equal(t, len(outRecords), 1)
	assert.Equal(t, outRecords[0].Face, inFace)
	assert.Equal(t, outRecords[0].LatestEncInterest, interest)
	assert.True(t, outRecords[0].LatestEncNonce == *interest.EncPacket.Interest.NonceV)

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
	assert.Equal(t, outRecords[0].LatestEncInterest, interest)
	assert.True(t, outRecords[0].LatestEncNonce == *interest.EncPacket.Interest.NonceV)

	assert.Equal(t, outRecords[1].Face, inFace2)
	assert.Equal(t, outRecords[1].LatestEncInterest, interest)
	assert.True(t, outRecords[1].LatestEncNonce == *interest.EncPacket.Interest.NonceV)
}

func TestInsertData(t *testing.T) {
	csReplacementPolicy = "lru"
	csCapacity = 1024
	pitCS := NewPitCS(func(PitEntry) {})

	// Data does not already exist
	name1, _ := enc.NameFromStr("/interest1")
	interest1 := makeInterest(name1)
	interest1.EncPacket.Interest.CanBePrefixV = false
	content := enc.Wire{enc.Buffer("test")}
	data1 := makeData(name1, content)

	pitCS.InsertData(data1)
	csEntry1 := pitCS.FindMatchingDataFromCS(interest1)
	assert.Equal(t, pitCS.CsSize(), 1)
	assert.True(t, bytes.Equal(csEntry1.EncData().EncPacket.Data.ContentV.Join(), data1.EncPacket.Data.ContentV.Join()))

	// Insert data associated with same name, so we should just update it
	// Should not result in a new CsEntry
	pitCS.InsertData(data1)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.Equal(t, pitCS.CsSize(), 1)
	assert.True(t, bytes.Equal(csEntry1.EncData().EncPacket.Data.ContentV.Join(), data1.EncPacket.Data.ContentV.Join()))

	// Insert some different data, should result in creation of new CsEntry
	name2, _ := enc.NameFromStr("/interest2")
	interest2 := makeInterest(name2)
	interest2.EncPacket.Interest.CanBePrefixV = false
	content2 := enc.Wire{enc.Buffer("test2")}
	data2 := makeData(name2, content2)
	pitCS.InsertData(data2)

	csEntry2 := pitCS.FindMatchingDataFromCS(interest2)
	assert.Equal(t, pitCS.CsSize(), 2)
	assert.True(t, bytes.Equal(csEntry2.EncData().EncPacket.Data.ContentV.Join(), data2.EncPacket.Data.ContentV.Join()))

	// PitCS with interest /a/b, data /a/b/v=10
	// Interest /a/b is prefix allowed
	// Should return data with name /a/b/v=10
	pitCS = NewPitCS(func(PitEntry) {})

	name1, _ = enc.NameFromStr("/a/b")
	interest1 = makeInterest(name1)
	interest1.EncPacket.Interest.CanBePrefixV = true
	name2, _ = enc.NameFromStr("/a/b/v=10")
	data2 = makeData(name2, content2)
	pitCS.InsertData(data2)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.Equal(t, pitCS.CsSize(), 1)
	assert.True(t, bytes.Equal(csEntry1.EncData().EncPacket.Data.ContentV.Join(), data2.EncPacket.Data.ContentV.Join()))

	// Reduced CS capacity to check that eviction occurs
	csCapacity = 1
	pitCS = NewPitCS(func(PitEntry) {})
	data1 = makeData(name1, content)
	data2 = makeData(name2, content2)
	pitCS.InsertData(data1)
	pitCS.InsertData(data2)
	assert.Equal(t, pitCS.CsSize(), 1)
}

func FindMatchingDataFromCS(t *testing.T) {
	csReplacementPolicy = "lru"
	csCapacity = 1024
	pitCS := NewPitCS(func(PitEntry) {})

	// Insert data and then fetch it
	name1, _ := enc.NameFromStr("/interest1")
	interest1 := makeInterest(name1)
	interest1.EncPacket.Interest.CanBePrefixV = false
	content := enc.Wire{enc.Buffer("test")}
	data1 := makeData(name1, content)

	pitCS.InsertData(data1)
	csEntry1 := pitCS.FindMatchingDataFromCS(interest1)
	assert.True(t, bytes.Equal(csEntry1.EncData().EncPacket.Data.ContentV.Join(), data1.EncPacket.Data.ContentV.Join()))

	// Insert some different data
	name2, _ := enc.NameFromStr("/interest2")
	interest2 := makeInterest(name2)
	interest2.EncPacket.Interest.CanBePrefixV = false
	content2 := enc.Wire{enc.Buffer("test2")}
	data2 := makeData(name2, content2)
	pitCS.InsertData(data2)

	csEntry2 := pitCS.FindMatchingDataFromCS(interest2)
	assert.True(t, bytes.Equal(csEntry2.EncData().EncPacket.Data.ContentV.Join(), data2.EncPacket.Data.ContentV.Join()))

	// Look for nonexistent data
	nameNonExistent, _ := enc.NameFromStr("/doesnotexist")
	interestNonExistent := makeInterest(nameNonExistent)
	csEntryNonExistent := pitCS.FindMatchingDataFromCS(interestNonExistent)
	assert.Nil(t, csEntryNonExistent)

	// PitCS with interest /a/b, data /a/b/v=10
	// Interest /a/b is prefix allowed
	// Should return data with name /a/b/v=10
	pitCS = NewPitCS(func(PitEntry) {})

	name1, _ = enc.NameFromStr("/a/b")
	interest1 = makeInterest(name1)
	interest1.EncPacket.Interest.CanBePrefixV = true
	name2, _ = enc.NameFromStr("/a/b/v=10")
	data2 = makeData(name2, content2)

	pitCS.InsertData(data2)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.True(t, bytes.Equal(csEntry1.EncData().EncPacket.Data.ContentV.Join(), data2.EncPacket.Data.ContentV.Join()))
	// PitCS with interest /a/b
	// Now look for interest /a/b with prefix allowed
	// Should return nil since there is no data
	pitCS = NewPitCS(func(PitEntry) {})

	name1, _ = enc.NameFromStr("/a/b")
	interest1 = makeInterest(name1)
	interest2.EncPacket.Interest.CanBePrefixV = true

	pitCS.InsertInterest(interest1, nil, 0)
	csEntry1 = pitCS.FindMatchingDataFromCS(interest1)
	assert.Nil(t, csEntry1)
}
