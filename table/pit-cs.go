/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/eric135/YaNFD/ndn"
)

// PitCs represents the PIT-CS tree for a thread.
type PitCs struct {
	root               *PitCsNode
	ExpiringPitEntries chan *PitEntry

	pitTokenMap map[uint32]*PitEntry

	nPitEntries int // Number of PIT entries in tree
	nCsEntries  int // Number of CS entries in tree
}

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
	node  *PitCsNode
	pitCs *PitCs

	Name           *ndn.Name
	CanBePrefix    bool
	MustBeFresh    bool
	ForwardingHint *ndn.Delegation          // Interests must match in terms of Forwarding Hint to be aggregated in PIT.
	InRecords      map[uint64]*PitInRecord  // Key is face ID
	OutRecords     map[uint64]*PitOutRecord // Key is face ID
	ExpirationTime time.Time
	Satisfied      bool

	Token uint32
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
type CsEntry struct {
	node *PitCsNode

	Data      *ndn.Data
	StaleTime time.Time
}

// NewPitCS creates a new combined PIT-CS for a forwarding thread.
func NewPitCS() *PitCs {
	pit := new(PitCs)
	pit.root = new(PitCsNode)
	pit.root.component = nil // Root component will be nil since it represents zero components
	pit.root.pitEntries = make([]*PitEntry, 0)
	pit.ExpiringPitEntries = make(chan *PitEntry, tableQueueSize)
	pit.pitTokenMap = make(map[uint32]*PitEntry)
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

func (p *PitCsNode) pruneIfEmpty() {
	for curNode := p; curNode.parent != nil && len(curNode.children) == 0 && len(curNode.pitEntries) == 0 && curNode.csEntry == nil; curNode = curNode.parent {
		// Remove from parent's children
		for i, child := range curNode.parent.children {
			if child == p {
				if i < len(curNode.parent.children)-1 {
					copy(curNode.parent.children[i:], curNode.parent.children[i+1:])
				}
				curNode.parent.children = curNode.parent.children[:len(curNode.parent.children)-1]
				break
			}
		}
	}
}

func (p *PitCs) generateNewPitToken() uint32 {
	for {
		token := rand.Uint32()
		if _, ok := p.pitTokenMap[token]; !ok {
			return token
		}
	}
}

// PitSize returns the number of entries in the PIT.
func (p *PitCs) PitSize() int {
	return p.nPitEntries
}

// CsSize returns the number of entries in the CS.
func (p *PitCs) CsSize() int {
	return p.nCsEntries
}

// FindOrInsertPIT inserts an entry in the PIT upon receipt of an Interest. Returns tuple of PIT entry and whether the Nonce is a duplicate.
func (p *PitCs) FindOrInsertPIT(interest *ndn.Interest, hint *ndn.Delegation, inFace uint64) (*PitEntry, bool) {
	node := p.root.fillTreeToPrefix(interest.Name())

	var entry *PitEntry
	for _, curEntry := range node.pitEntries {
		if curEntry.CanBePrefix == interest.CanBePrefix() && curEntry.MustBeFresh == interest.MustBeFresh() && ((hint == nil && curEntry.ForwardingHint == nil) || hint.Name().Equals(curEntry.ForwardingHint.Name())) {
			entry = curEntry
			break
		}
	}

	if entry == nil {
		p.nPitEntries++
		entry = new(PitEntry)
		entry.node = node
		entry.pitCs = p
		entry.Name = interest.Name()
		entry.CanBePrefix = interest.CanBePrefix()
		entry.MustBeFresh = interest.MustBeFresh()
		entry.ForwardingHint = hint
		entry.InRecords = make(map[uint64]*PitInRecord, 0)
		entry.OutRecords = make(map[uint64]*PitOutRecord, 0)
		entry.Satisfied = false
		node.pitEntries = append(node.pitEntries, entry)
		entry.Token = p.generateNewPitToken()
		p.pitTokenMap[entry.Token] = entry
	}

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
func (p *PitCs) FindPITFromData(data *ndn.Data, token uint32) *PitEntry {
	if entry, ok := p.pitTokenMap[token]; ok && entry.Token == token {
		return entry
	}
	return nil
}

// FindPITFromToken finds the PIT entry indexed by a given token or nil if none were found.
func (p *PitCs) FindPITFromToken(token uint32) *PitEntry {
	if entry, ok := p.pitTokenMap[token]; ok {
		return entry
	}
	return nil
}

// RemovePITEntry removes the specified PIT entry.
func (p *PitCs) RemovePITEntry(pitEntry *PitEntry) bool {
	node := p.root.findExactMatchEntry(pitEntry.Name)
	if node != nil {
		for i, entry := range node.pitEntries {
			if entry == pitEntry {
				if i < len(node.pitEntries)-1 {
					copy(node.pitEntries[i:], node.pitEntries[i+1:])
				}
				node.pitEntries = node.pitEntries[:len(node.pitEntries)-1]
				if len(node.pitEntries) == 0 {
					entry.node.pruneIfEmpty()
				}
				return true
			}
		}
	}
	return false
}

// FindMatchingDataCS finds the best matching entry in the CS (if any). If MustBeFresh is set to true in the Interest, only non-stale CS entries will be returned.
func (p *PitCs) FindMatchingDataCS(interest *ndn.Interest) *CsEntry {
	node := p.root.findExactMatchEntry(interest.Name())
	if node != nil {
		if !interest.CanBePrefix() {
			return node.csEntry
		}
		return node.findMatchingDataCSPrefix(interest)
	}
	return nil
}

// InsertDataCS inserts a Data packet into the Content Store.
func (p *PitCs) InsertDataCS(data *ndn.Data) {
	// TODO: Eviction if needed

	node := p.root.fillTreeToPrefix(data.Name())
	if node.csEntry != nil {
		// Replace
		p.nPitEntries++
		node.csEntry.Data = data

		// TODO: Remove some entries from PIT if needed

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

func (e *PitEntry) waitForPitExpiry() {
	if !e.ExpirationTime.IsZero() {
		time.Sleep(e.ExpirationTime.Sub(time.Now().Add(time.Millisecond * 1))) // Add 1 millisecond to ensure *after* expiration
		if e.ExpirationTime.Before(time.Now()) {
			// Otherwise, has been updated by another PIT entry
			e.pitCs.ExpiringPitEntries <- e
		}
	}
}

// FindOrInsertInRecord finds or inserts an InRecord for the face, updating the metadata and returning whether there was already an in-record in the entry.
func (e *PitEntry) FindOrInsertInRecord(interest *ndn.Interest, face uint64) (*PitInRecord, bool) {
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
func (e *PitEntry) FindOrInsertOutRecord(interest *ndn.Interest, face uint64) *PitOutRecord {
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
	e.ExpirationTime = time.Unix(0, 0)

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

	go e.waitForPitExpiry()
}

// SetExpirationTimerToNow updates the expiration timer to the current time.
func (e *PitEntry) SetExpirationTimerToNow() {
	e.ExpirationTime = time.Now()
	e.pitCs.ExpiringPitEntries <- e
}

// ClearInRecords removes all in-records from the PIT entry.
func (e *PitEntry) ClearInRecords() {
	e.InRecords = make(map[uint64]*PitInRecord)
}

// ClearOutRecords removes all out-records from the PIT entry.
func (e *PitEntry) ClearOutRecords() {
	e.OutRecords = make(map[uint64]*PitOutRecord)
}
