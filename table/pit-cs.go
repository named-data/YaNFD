/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"bytes"
	"time"

	"github.com/eric135/YaNFD/ndn"
)

// PitCsNode represents an entry in a PIT-CS tree.
type PitCsNode struct {
	component ndn.NameComponent
	depth     int

	parent   *PitCsNode
	children []*PitCsNode

	pitEntries []*PitEntry
	csEntry    *CsEntry
}

// PitEntry is an entry in a thread's PIT.
type PitEntry struct {
	node *PitCsNode

	name        *ndn.Name
	CanBePrefix bool
	MustBeFresh bool
	//forwardingHint *ndn.ForwardingHint // TODO: Interests must match in terms of Forwarding Hint to be aggregated in PIT.
	InRecords      map[int]*PitInRecord  // Key is face ID
	OutRecords     map[int]*PitOutRecord // Key is face ID
	ExpirationTime time.Time
}

// PitInRecord records an incoming Interest on a given face.
type PitInRecord struct {
	Face            int
	LatestNonce     []byte
	LatestTimestamp time.Time
	LatestInterest  *ndn.Interest
	ExpirationTime  time.Time
	PitToken        []byte
}

// PitOutRecord records an outgoing Interest on a given face.
type PitOutRecord struct {
	Face            int
	LatestNonce     []byte
	LatestTimestamp time.Time
	LatestInterest  *ndn.Interest
	ExpirationTime  time.Time
}

// CsEntry is an entry in a thread's CS.
type CsEntry struct {
	node *PitCsNode

	Data      *ndn.Data
	StaleTime time.Time
}

// NewPitCS creates a new combined PIT-CS for a forwarding thread.
func NewPitCS() *PitCsNode {
	pit := new(PitCsNode)
	pit.component = nil // Root component will be nil since it represents zero components
	pit.pitEntries = make([]*PitEntry, 0)
	return pit
}

func (p *PitCsNode) findExactMatchEntry(name *ndn.Name) *PitCsNode {
	if name.Size() > p.depth {
		for _, child := range p.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findExactMatchEntry(name)
			}
		}
	} else if name.Size() == p.depth {
		return p
	}
	return nil
}

func (p *PitCsNode) findLongestPrefixEntry(name *ndn.Name) *PitCsNode {
	if name.Size() > p.depth {
		for _, child := range p.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findLongestPrefixEntry(name)
			}
		}
	}
	return p
}

func (p *PitCsNode) fillTreeToPrefix(name *ndn.Name) *PitCsNode {
	curNode := p.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(PitCsNode)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		curNode.children = append(curNode.children, newNode)
		curNode = newNode
	}
	return curNode
}

// FindOrInsertPIT inserts an entry in the PIT upon receipt of an Interest. Returns tuple of PIT entry and whether the Nonce is a duplicate.
func (p *PitCsNode) FindOrInsertPIT(interest *ndn.Interest, inFace int) (*PitEntry, bool) {
	node := p.fillTreeToPrefix(interest.Name())

	var entry *PitEntry
	for _, curEntry := range node.pitEntries {
		// TODO: ForwardingHint
		if curEntry.CanBePrefix == interest.CanBePrefix() && curEntry.MustBeFresh == interest.MustBeFresh() {
			entry = curEntry
			break
		}
	}

	if entry == nil {
		entry = new(PitEntry)
		entry.node = node
		entry.name = interest.Name()
		entry.CanBePrefix = interest.CanBePrefix()
		entry.MustBeFresh = interest.MustBeFresh()
		// TODO: ForwardingHint
		entry.InRecords = make(map[int]*PitInRecord, 0)
		entry.OutRecords = make(map[int]*PitOutRecord, 0)
		node.pitEntries = append(node.pitEntries, entry)
	}

	// Lazily erase expired records.
	entry.lazilyEraseExpiredPITRecords()

	for face, inRecord := range entry.InRecords {
		// Only considered a duplicate (loop) if from different face since is just retransmission and not loop if same face
		if face != inFace && bytes.Equal(inRecord.LatestNonce, interest.Nonce()) {
			return entry, true
		}
	}

	// Cancel expiration time
	entry.ExpirationTime = time.Unix(0, 0)

	return entry, false
}

// FindPITFromData finds the PIT entries matching a Data packet. Note that this does not consider the effect of MustBeFresh.
func (p *PitCsNode) FindPITFromData(data *ndn.Data) []*PitEntry {
	matching := make([]*PitEntry, 0)
	dataNameLen := data.Name().Size()
	for curNode := p.findLongestPrefixEntry(data.Name()); curNode != nil; curNode = curNode.parent {
		for _, entry := range curNode.pitEntries {
			if entry.CanBePrefix || curNode.depth == dataNameLen {
				matching = append(matching, entry)
			}
		}
	}
	return matching
}

// FindMatchingDataCS finds the best matching entry in the CS (if any). If MustBeFresh is set to true in the Interest, only non-stale CS entries will be returned.
func (p *PitCsNode) FindMatchingDataCS(interest *ndn.Interest) *CsEntry {
	node := p.findExactMatchEntry(interest.Name())
	if node != nil {
		if !interest.CanBePrefix() {
			return node.csEntry
		}
		return node.findMatchingDataCSPrefix(interest)
	}
	return nil
}

// InsertDataCS inserts a Data packet into the Content Store.
func (p *PitCsNode) InsertDataCS(data *ndn.Data) {
	// TODO: Eviction if needed

	node := p.fillTreeToPrefix(data.Name())
	if node.csEntry != nil {
		// Replace
		node.csEntry.Data = data
		if data.MetaInfo() == nil || data.MetaInfo().FinalBlockID() == nil {
			node.csEntry.StaleTime = time.Now()
		} else {
			node.csEntry.StaleTime = time.Now().Add(*data.MetaInfo().FreshnessPeriod())
		}
	}
}

func (p *PitCsNode) findMatchingDataCSPrefix(interest *ndn.Interest) *CsEntry {
	if p.csEntry != nil && (!interest.MustBeFresh() || time.Now().Before(p.csEntry.StaleTime)) {
		return p.csEntry
	}

	if p.depth < interest.Name().Size() {
		for _, child := range p.children {
			if interest.Name().At(p.depth).Equals(child.component) {
				return child.findMatchingDataCSPrefix(interest)
			}
		}
	}

	// If found none, then return
	return nil
}

func (e *PitEntry) lazilyEraseExpiredPITRecords() {
	now := time.Now()

	for _, inRecord := range e.InRecords {
		if inRecord.LatestTimestamp.Add(inRecord.LatestInterest.Lifetime()).Before(now) {
			delete(e.InRecords, inRecord.Face)
		}
	}

	for _, outRecord := range e.OutRecords {
		if outRecord.LatestTimestamp.Add(outRecord.LatestInterest.Lifetime()).Before(now) {
			delete(e.InRecords, outRecord.Face)
		}
	}
}

// FindOrInsertInRecord finds or inserts an InRecord for the face, updating the metadata and returning whether there was already an in-record in the entry.
func (e *PitEntry) FindOrInsertInRecord(interest *ndn.Interest, face int) (*PitInRecord, bool) {
	// Lazily erase expired records
	e.lazilyEraseExpiredPITRecords()

	var record *PitInRecord
	var ok bool
	if record, ok = e.InRecords[face]; !ok {
		record := new(PitInRecord)
		record.Face = face
		record.LatestNonce = interest.Nonce()
		record.LatestTimestamp = time.Now()
		record.LatestInterest = interest
		record.ExpirationTime = time.Now().Add(interest.Lifetime())
		e.InRecords[face] = record
		return record, len(e.InRecords) > 1
	}

	// Existing record
	record.LatestNonce = interest.Nonce()
	record.LatestTimestamp = time.Now()
	record.LatestInterest = interest
	record.ExpirationTime = time.Now().Add(interest.Lifetime())
	return record, true
}

// FindOrInsertOutRecord finds or inserts an OutRecord for the face, updating the metadata.
func (e *PitEntry) FindOrInsertOutRecord(interest *ndn.Interest, face int) *PitOutRecord {
	// Lazily erase expired records
	e.lazilyEraseExpiredPITRecords()

	var record *PitOutRecord
	var ok bool
	if record, ok = e.OutRecords[face]; !ok {
		record := new(PitOutRecord)
		record.Face = face
		record.LatestNonce = interest.Nonce()
		record.LatestTimestamp = time.Now()
		record.LatestInterest = interest
		record.ExpirationTime = time.Now().Add(interest.Lifetime())
		e.OutRecords[face] = record
		return record
	}

	// Existing record
	record.LatestNonce = interest.Nonce()
	record.LatestTimestamp = time.Now()
	record.LatestInterest = interest
	record.ExpirationTime = time.Now().Add(interest.Lifetime())
	return record
}

// UpdateExpirationTimer updates the expiration timer to the latest expiration time of any in or out record in the entry.
func (e *PitEntry) UpdateExpirationTimer() {
	// Assumption: expiration time already reset to 0 Unix seconds.
	for _, record := range e.InRecords {
		if record.ExpirationTime.After(e.ExpirationTime) {
			e.ExpirationTime = record.ExpirationTime
		}
	}

	for _, record := range e.OutRecords {
		if record.ExpirationTime.After(e.ExpirationTime) {
			e.ExpirationTime = record.ExpirationTime
		}
	}
}

// SetExpirationTimerToNow updates the expiration timer to the current time.
func (e *PitEntry) SetExpirationTimerToNow() {
	e.ExpirationTime = time.Now()
}

// ClearInRecords removes all in-records from the PIT entry.
func (e *PitEntry) ClearInRecords() {
	e.InRecords = make(map[int]*PitInRecord)
}

// ClearOutRecords removes all out-records from the PIT entry.
func (e *PitEntry) ClearOutRecords() {
	e.OutRecords = make(map[int]*PitOutRecord)
}
