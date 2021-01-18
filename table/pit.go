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

// PitNode represents an entry in a PIT tree.
type PitNode struct {
	component ndn.NameComponent
	depth     int

	parent   *PitNode
	children []*PitNode

	entries []*PitEntry
}

// PitEntry is an entry in a thread's PIT.
type PitEntry struct {
	node *PitNode

	name        *ndn.Name
	canBePrefix bool
	mustBeFresh bool
	//forwardingHint *ndn.ForwardingHint // TODO: Interests must match in terms of Forwarding Hint to be aggregated in PIT.
	inRecords  map[int]*PitInRecord  // Key is face ID
	outRecords map[int]*PitOutRecord // Key is face ID
	// TODO: Timer
}

// PitInRecord records an incoming Interest on a given face.
type PitInRecord struct {
	face            int
	latestNonce     []byte
	latestTimestamp time.Time
	latestInterest  *ndn.Interest
}

// PitOutRecord records an outgoing Interest on a given face.
type PitOutRecord struct {
	face            int
	latestNonce     []byte
	latestTimestamp time.Time
	latestInterest  *ndn.Interest
}

// NewPit creates a new Pending Interest Table for a forwarding thread.
func NewPit() *PitNode {
	pit := new(PitNode)
	pit.component = nil // Root component will be nil since it represents zero components
	pit.entries = make([]*PitEntry, 0)
	return pit
}

func (p *PitNode) findLongestPrefixEntry(name *ndn.Name) *PitNode {
	if name.Size() > p.depth {
		for _, child := range p.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findLongestPrefixEntry(name)
			}
		}
	}
	return p
}

func (p *PitNode) fillTreeToPrefix(name *ndn.Name) *PitNode {
	curNode := p.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(PitNode)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		curNode.children = append(curNode.children, newNode)
		curNode = newNode
	}
	return curNode
}

// FindOrInsert inserts an entry in the PIT upon receipt of an Interest. Returns tuple of PIT entry and whether the Interest is a duplicate.
func (p *PitNode) FindOrInsert(interest *ndn.Interest, inFace int) (*PitEntry, bool) {
	node := p.fillTreeToPrefix(interest.Name())

	var entry *PitEntry
	for _, curEntry := range node.entries {
		// TODO: ForwardingHint
		if curEntry.canBePrefix == interest.CanBePrefix() && curEntry.mustBeFresh == interest.MustBeFresh() {
			entry = curEntry
			break
		}
	}

	if entry == nil {
		entry = new(PitEntry)
		entry.node = node
		entry.name = interest.Name()
		entry.canBePrefix = interest.CanBePrefix()
		entry.mustBeFresh = interest.MustBeFresh()
		// TODO: ForwardingHint
		entry.inRecords = make(map[int]*PitInRecord, 0)
		entry.outRecords = make(map[int]*PitOutRecord, 0)
		node.entries = append(node.entries, entry)
	}

	// Lazily erase expired records.
	entry.lazilyEraseExpiredRecords()

	var inRecord *PitInRecord
	if inRecord, ok := entry.inRecords[inFace]; !ok {
		inRecord = new(PitInRecord)
		inRecord.face = inFace
		entry.inRecords[inFace] = inRecord
	}
	oldNonce := inRecord.latestNonce
	inRecord.latestNonce = interest.Nonce()
	inRecord.latestTimestamp = time.Now()
	inRecord.latestInterest = interest

	return entry, bytes.Equal(oldNonce, inRecord.latestNonce)
}

// FindFromData find the PIT entries matching a Data packet. Note that this does not consider the effect of MustBeFresh.
func (p *PitNode) FindFromData(data *ndn.Data) []*PitEntry {
	matching := make([]*PitEntry, 0)
	dataNameLen := data.Name().Size()
	for curNode := p.findLongestPrefixEntry(data.Name()); curNode != nil; curNode = curNode.parent {
		for _, entry := range curNode.entries {
			if entry.canBePrefix || curNode.depth == dataNameLen {
				matching = append(matching, entry)
			}
		}
	}
	return matching
}

func (e *PitEntry) lazilyEraseExpiredRecords() {
	now := time.Now()

	for _, inRecord := range e.inRecords {
		if inRecord.latestTimestamp.Add(inRecord.latestInterest.Lifetime()).Before(now) {
			delete(e.inRecords, inRecord.face)
		}
	}

	for _, outRecord := range e.outRecords {
		if outRecord.latestTimestamp.Add(outRecord.latestInterest.Lifetime()).Before(now) {
			delete(e.inRecords, outRecord.face)
		}
	}
}
