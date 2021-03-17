/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
)

// PitCs represents the PIT-CS tree for a thread.
type PitCs struct {
	root               *PitCsNode
	ExpiringPitEntries chan *PitEntry

	nPitEntries int // Number of PIT entries in tree
	pitTokenMap map[uint32]*PitEntry

	nCsEntries    int // Number of CS entries in tree
	csReplacement CsReplacementPolicy
	csMap         map[uint64]*CsEntry
}

// PitCsNode represents an entry in a PIT-CS tree.
type PitCsNode struct {
	component ndn.NameComponent
	depth     int

	parent   *PitCsNode
	children []*PitCsNode

	pitEntries []*PitEntry

	csEntry *CsEntry
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
	node  *PitCsNode
	index uint64

	Data      *ndn.Data
	StaleTime time.Time
}

// NewPitCS creates a new combined PIT-CS for a forwarding thread.
func NewPitCS() *PitCs {
	pitCs := new(PitCs)
	pitCs.root = new(PitCsNode)
	pitCs.root.component = nil // Root component will be nil since it represents zero components
	pitCs.root.pitEntries = make([]*PitEntry, 0)
	pitCs.ExpiringPitEntries = make(chan *PitEntry, tableQueueSize)
	pitCs.pitTokenMap = make(map[uint32]*PitEntry)

	// This value has already been validated from loading the configuration, so we know it will be one of the following (or else fatal)
	switch csReplacementPolicy {
	case "lru":
		pitCs.csReplacement = NewCsLRU(pitCs)
	default:
		core.LogFatal(pitCs, "Unknown CS replacement policy "+csReplacementPolicy)
	}
	pitCs.csMap = make(map[uint64]*CsEntry)

	return pitCs
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

func (p *PitCs) hashCsName(name *ndn.Name) uint64 {
	sum := sha512.Sum512([]byte(name.String()))
	return binary.BigEndian.Uint64(sum[56:])
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
		entry.InRecords = make(map[uint64]*PitInRecord)
		entry.OutRecords = make(map[uint64]*PitOutRecord)
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
func (p *PitCs) FindPITFromData(data *ndn.Data, token *uint32) []*PitEntry {
	if token != nil {
		if entry, ok := p.pitTokenMap[*token]; ok && entry.Token == *token {
			return []*PitEntry{entry}
		}
		return nil
	}

	matching := make([]*PitEntry, 0)
	dataNameLen := data.Name().Size()
	for curNode := p.root.findLongestPrefixEntry(data.Name()); curNode != nil; curNode = curNode.parent {
		for _, entry := range curNode.pitEntries {
			if entry.CanBePrefix || curNode.depth == dataNameLen {
				matching = append(matching, entry)
			}
		}
	}
	return matching
}

// RemovePITEntry removes the specified PIT entry.
func (p *PitCs) RemovePITEntry(pitEntry *PitEntry) bool {
	for i, entry := range pitEntry.node.pitEntries {
		if entry == pitEntry {
			if i < len(pitEntry.node.pitEntries)-1 {
				copy(pitEntry.node.pitEntries[i:], pitEntry.node.pitEntries[i+1:])
			}
			pitEntry.node.pitEntries = pitEntry.node.pitEntries[:len(pitEntry.node.pitEntries)-1]
			if len(pitEntry.node.pitEntries) == 0 {
				entry.node.pruneIfEmpty()
			}
			p.nPitEntries--
			return true
		}
	}
	return false
}

// FindMatchingDataCS finds the best matching entry in the CS (if any). If MustBeFresh is set to true in the Interest, only non-stale CS entries will be returned.
func (p *PitCs) FindMatchingDataCS(interest *ndn.Interest) *CsEntry {
	node := p.root.findExactMatchEntry(interest.Name())
	if node != nil {
		if !interest.CanBePrefix() {
			if node.csEntry != nil {
				p.csReplacement.BeforeUse(node.csEntry.index, node.csEntry.Data)
			}
			return node.csEntry
		}
		return node.findMatchingDataCSPrefix(interest)
	}
	return nil
}

// InsertDataCS inserts a Data packet into the Content Store.
func (p *PitCs) InsertDataCS(data *ndn.Data) {
	index := p.hashCsName(data.Name())

	if entry, ok := p.csMap[index]; ok {
		// Replace existing entry
		entry.Data = data

		if data.MetaInfo() == nil || data.MetaInfo().FinalBlockID() == nil {
			entry.StaleTime = time.Now()
		} else {
			entry.StaleTime = time.Now().Add(*data.MetaInfo().FreshnessPeriod())
		}

		p.csReplacement.AfterRefresh(index, data)
	} else {
		// New entry
		p.nCsEntries++
		node := p.root.fillTreeToPrefix(data.Name())
		node.csEntry = new(CsEntry)
		node.csEntry.node = node
		node.csEntry.index = index
		node.csEntry.Data = data
		p.csMap[index] = node.csEntry
		p.csReplacement.AfterInsert(index, data)

		// Tell replacement strategy to evict entries if needed
		p.csReplacement.EvictEntries()
	}
}

// eraseCsDataFromReplacementStrategy allows the replacement strategy to erase the data with the specified name from the Content Store.
func (p *PitCs) eraseCsDataFromReplacementStrategy(index uint64) {
	if entry, ok := p.csMap[index]; ok {
		entry.node.csEntry = nil
		delete(p.csMap, index)
		p.nCsEntries--
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
func (e *PitEntry) FindOrInsertInRecord(interest *ndn.Interest, face uint64, incomingPitToken []byte) (*PitInRecord, bool) {
	var record *PitInRecord
	var ok bool
	if record, ok = e.InRecords[face]; !ok {
		record := new(PitInRecord)
		record.Face = face
		record.LatestNonce = interest.Nonce()
		record.LatestTimestamp = time.Now()
		record.LatestInterest = interest
		record.ExpirationTime = time.Now().Add(interest.Lifetime())
		record.PitToken = incomingPitToken
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
