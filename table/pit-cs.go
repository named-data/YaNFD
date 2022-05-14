/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"time"

	"github.com/named-data/YaNFD/ndn"
)

// PitCsTable dictates what functionality a Pit-Cs table should implement
// Warning: All functions must be called in the same forwarding goroutine as the creation of the table.
type PitCsTable interface {
	InsertInterest(interest *ndn.Interest, hint *ndn.Name, inFace uint64) (PitEntry, bool)
	RemoveInterest(pitEntry PitEntry) bool
	FindInterestExactMatch(interest *ndn.Interest) PitEntry
	FindInterestPrefixMatchByData(data *ndn.Data, token *uint32) []PitEntry
	PitSize() int

	InsertData(data *ndn.Data)
	FindMatchingDataFromCS(interest *ndn.Interest) CsEntry
	CsSize() int
	IsCsAdmitting() bool
	IsCsServing() bool

	eraseCsDataFromReplacementStrategy(index uint64)
	updatePitExpiry(pitEntry PitEntry)

	// UpdateTimer returns the channel used to signal regular Update() calls in the forwarding thread.
	// <- UpdateTimer() and Update() must be called in pairs.
	UpdateTimer() <-chan struct{}
	// Update() does whatever the PIT table needs to do regularly.
	// It may schedule the next UpdateTimer().
	Update()
}

// basePitCsTable contains properties common to all PIT-CS tables
type basePitCsTable struct{}

// PitEntry dictates what entries in a PIT-CS table should implement
type PitEntry interface {
	PitCs() PitCsTable
	Name() *ndn.Name
	CanBePrefix() bool
	MustBeFresh() bool
	ForwardingHint() *ndn.Name
	// Interests must match in terms of Forwarding Hint to be aggregated in PIT.
	InRecords() map[uint64]*PitInRecord   // Key is face ID
	OutRecords() map[uint64]*PitOutRecord // Key is face ID
	ExpirationTime() time.Time
	SetExpirationTime(t time.Time)
	Satisfied() bool
	SetSatisfied(isSatisfied bool)

	Token() uint32

	InsertInRecord(interest *ndn.Interest, face uint64, incomingPitToken []byte) (*PitInRecord, bool)
	InsertOutRecord(interest *ndn.Interest, face uint64) *PitOutRecord

	GetOutRecords() []*PitOutRecord
	ClearOutRecords()
	ClearInRecords()
}

// basePitEntry contains PIT entry properties common to all tables.
type basePitEntry struct {
	// lowercase fields so that they aren't exported
	name           *ndn.Name
	canBePrefix    bool
	mustBeFresh    bool
	forwardingHint *ndn.Name
	// Interests must match in terms of Forwarding Hint to be
	// aggregated in PIT.
	inRecords      map[uint64]*PitInRecord  // Key is face ID
	outRecords     map[uint64]*PitOutRecord // Key is face ID
	expirationTime time.Time
	satisfied      bool

	token uint32
}

// PitInRecord records an incoming Interest on a given face.
type PitInRecord struct {
	Face            uint64
	LatestNonce     []byte
	LatestTimestamp time.Time
	LatestInterest  *ndn.Interest
	ExpirationTime  time.Time
	PitToken        []byte
}

// PitOutRecord records an outgoing Interest on a given face.
type PitOutRecord struct {
	Face            uint64
	LatestNonce     []byte
	LatestTimestamp time.Time
	LatestInterest  *ndn.Interest
	ExpirationTime  time.Time
}

// CsEntry is an entry in a thread's CS.
type CsEntry interface {
	Index() uint64 // the hash of the entry, for fast lookup
	StaleTime() time.Time
	Data() *ndn.Data
}

type baseCsEntry struct {
	index     uint64
	staleTime time.Time
	data      *ndn.Data
}

// InsertInRecord finds or inserts an InRecord for the face, updating the
// metadata and returning whether there was already an in-record in the entry.
func (bpe *basePitEntry) InsertInRecord(interest *ndn.Interest, face uint64, incomingPitToken []byte) (*PitInRecord, bool) {
	var record *PitInRecord
	var ok bool
	if record, ok = bpe.inRecords[face]; !ok {
		record := new(PitInRecord)
		record.Face = face
		record.LatestNonce = interest.Nonce()
		record.LatestTimestamp = time.Now()
		record.LatestInterest = interest
		record.ExpirationTime = time.Now().Add(interest.Lifetime())
		record.PitToken = incomingPitToken
		bpe.inRecords[face] = record
		return record, false
	}

	// Existing record
	record.LatestNonce = interest.Nonce()
	record.LatestTimestamp = time.Now()
	record.LatestInterest = interest
	record.ExpirationTime = time.Now().Add(interest.Lifetime())
	return record, true
}

// SetExpirationTimerToNow updates the expiration timer to the current time.
func SetExpirationTimerToNow(e PitEntry) {
	e.SetExpirationTime(time.Now())
	e.PitCs().updatePitExpiry(e)
}

// UpdateExpirationTimer updates the expiration timer to the latest expiration
// time of any in or out record in the entry.
func UpdateExpirationTimer(e PitEntry) {
	e.SetExpirationTime(time.Now())

	for _, record := range e.InRecords() {
		if record.ExpirationTime.After(e.ExpirationTime()) {
			e.SetExpirationTime(record.ExpirationTime)
		}
	}

	for _, record := range e.OutRecords() {
		if record.ExpirationTime.After(e.ExpirationTime()) {
			e.SetExpirationTime(record.ExpirationTime)
		}
	}

	e.PitCs().updatePitExpiry(e)
}

///// Setters and Getters /////

func (bpe *basePitEntry) Name() *ndn.Name {
	return bpe.name
}

func (bpe *basePitEntry) CanBePrefix() bool {
	return bpe.canBePrefix
}

func (bpe *basePitEntry) MustBeFresh() bool {
	return bpe.mustBeFresh
}

func (bpe *basePitEntry) ForwardingHint() *ndn.Name {
	return bpe.forwardingHint
}

func (bpe *basePitEntry) InRecords() map[uint64]*PitInRecord {
	return bpe.inRecords
}

func (bpe *basePitEntry) OutRecords() map[uint64]*PitOutRecord {
	return bpe.outRecords
}

// ClearInRecords removes all in-records from the PIT entry.
func (bpe *basePitEntry) ClearInRecords() {
	bpe.inRecords = make(map[uint64]*PitInRecord)
}

// ClearOutRecords removes all out-records from the PIT entry.
func (bpe *basePitEntry) ClearOutRecords() {
	bpe.outRecords = make(map[uint64]*PitOutRecord)
}

func (bpe *basePitEntry) ExpirationTime() time.Time {
	return bpe.expirationTime
}

func (bpe *basePitEntry) SetExpirationTime(t time.Time) {
	bpe.expirationTime = t
}

func (bpe *basePitEntry) Satisfied() bool {
	return bpe.satisfied
}

func (bpe *basePitEntry) SetSatisfied(isSatisfied bool) {
	bpe.satisfied = isSatisfied
}

func (bpe *basePitEntry) Token() uint32 {
	return bpe.token
}

func (bce *baseCsEntry) Index() uint64 {
	return bce.index
}

func (bce *baseCsEntry) StaleTime() time.Time {
	return bce.staleTime
}

func (bce *baseCsEntry) Data() *ndn.Data {
	return bce.data
}
