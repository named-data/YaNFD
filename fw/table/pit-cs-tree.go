package table

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/cespare/xxhash"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/utils/priority_queue"
)

const expiredPitTickerInterval = 100 * time.Millisecond

type OnPitExpiration func(PitEntry)

// PitCsTree represents a PIT-CS implementation that uses a name tree
type PitCsTree struct {
	basePitCsTable

	root *pitCsTreeNode

	nPitEntries int
	pitTokenMap map[uint32]*nameTreePitEntry

	nCsEntries    int
	csReplacement CsReplacementPolicy
	csMap         map[uint64]*nameTreeCsEntry

	pitExpiryQueue priority_queue.Queue[*nameTreePitEntry, int64]
	updateTimer    chan struct{}
	onExpiration   OnPitExpiration
}

type nameTreePitEntry struct {
	basePitEntry                // compose with BasePitEntry
	pitCsTable   *PitCsTree     // pointer to tree
	node         *pitCsTreeNode // the tree node associated with this entry
	queueIndex   int            // index of entry in the expiring queue
}

type nameTreeCsEntry struct {
	baseCsEntry                // compose with BasePitEntry
	node        *pitCsTreeNode // the tree node associated with this entry
}

// pitCsTreeNode represents an entry in a PIT-CS tree.
type pitCsTreeNode struct {
	component ndn.NameComponent
	depth     int

	parent   *pitCsTreeNode
	children map[string]*pitCsTreeNode

	pitEntries []*nameTreePitEntry

	csEntry *nameTreeCsEntry
}

// NewPitCS creates a new combined PIT-CS for a forwarding thread.
func NewPitCS(onExpiration OnPitExpiration) *PitCsTree {
	pitCs := new(PitCsTree)
	pitCs.root = new(pitCsTreeNode)
	pitCs.root.component = nil // Root component will be nil since it represents zero components
	pitCs.root.pitEntries = make([]*nameTreePitEntry, 0)
	pitCs.root.children = make(map[string]*pitCsTreeNode)
	pitCs.onExpiration = onExpiration
	pitCs.pitTokenMap = make(map[uint32]*nameTreePitEntry)
	pitCs.pitExpiryQueue = priority_queue.New[*nameTreePitEntry, int64]()
	pitCs.updateTimer = make(chan struct{})

	// This value has already been validated from loading the configuration, so we know it will be one of the following (or else fatal)
	switch csReplacementPolicy {
	case "lru":
		pitCs.csReplacement = NewCsLRU(pitCs)
	default:
		core.LogFatal(pitCs, "Unknown CS replacement policy ", csReplacementPolicy)
	}
	pitCs.csMap = make(map[uint64]*nameTreeCsEntry)

	// Schedule first signal
	time.AfterFunc(expiredPitTickerInterval, func() {
		pitCs.updateTimer <- struct{}{}
	})

	return pitCs
}

func (p *PitCsTree) UpdateTimer() <-chan struct{} {
	return p.updateTimer
}

func (p *PitCsTree) Update() {
	for p.pitExpiryQueue.Len() > 0 && p.pitExpiryQueue.PeekPriority() <= time.Now().UnixNano() {
		entry := p.pitExpiryQueue.Pop()
		entry.queueIndex = -1
		p.onExpiration(entry)
		p.RemoveInterest(entry)
	}
	if !core.ShouldQuit {
		updateDuration := expiredPitTickerInterval
		if p.pitExpiryQueue.Len() > 0 {
			sleepTime := time.Duration(p.pitExpiryQueue.PeekPriority()-time.Now().UnixNano()) * time.Nanosecond
			if sleepTime > 0 {
				if sleepTime > expiredPitTickerInterval {
					sleepTime = expiredPitTickerInterval
				}
				updateDuration = sleepTime
			}
		}
		// Schedule next signal
		time.AfterFunc(updateDuration, func() {
			p.updateTimer <- struct{}{}
		})
	}
}

func (p *PitCsTree) updatePitExpiry(pitEntry PitEntry) {
	e := pitEntry.(*nameTreePitEntry)
	if e.queueIndex < 0 {
		e.queueIndex = p.pitExpiryQueue.Push(e, e.expirationTime.UnixNano())
	} else {
		p.pitExpiryQueue.Update(e.queueIndex, e, e.expirationTime.UnixNano())
	}
}

func (e *nameTreePitEntry) PitCs() PitCsTable {
	return e.pitCsTable
}

// InsertInterest inserts an entry in the PIT upon receipt of an Interest.
// Returns tuple of PIT entry and whether the Nonce is a duplicate.
func (p *PitCsTree) InsertInterest(interest *ndn.Interest, hint *ndn.Name, inFace uint64) (PitEntry, bool) {
	node := p.root.fillTreeToPrefix(interest.Name())

	var entry *nameTreePitEntry
	for _, curEntry := range node.pitEntries {
		if curEntry.CanBePrefix() == interest.CanBePrefix() && curEntry.MustBeFresh() == interest.MustBeFresh() && ((hint == nil && curEntry.ForwardingHint() == nil) || hint.Equals(curEntry.ForwardingHint())) {
			entry = curEntry
			break
		}
	}

	if entry == nil {
		p.nPitEntries++
		entry = new(nameTreePitEntry)
		entry.node = node
		entry.pitCsTable = p
		entry.name = interest.Name()
		entry.canBePrefix = interest.CanBePrefix()
		entry.mustBeFresh = interest.MustBeFresh()
		entry.forwardingHint = hint
		entry.inRecords = make(map[uint64]*PitInRecord)
		entry.outRecords = make(map[uint64]*PitOutRecord)
		entry.satisfied = false
		node.pitEntries = append(node.pitEntries, entry)
		entry.token = p.generateNewPitToken()
		entry.queueIndex = -1
		p.pitTokenMap[entry.token] = entry
	}

	for face, inRecord := range entry.inRecords {
		// Only considered a duplicate (loop) if from different face since is just retransmission and not loop if same face
		if face != inFace && bytes.Equal(inRecord.LatestNonce, interest.Nonce()) {
			return entry, true
		}
	}

	// Cancel expiration time
	entry.expirationTime = time.Unix(0, 0)

	return entry, false
}

// RemoveInterest removes the specified PIT entry, returning true if the entry
// was removed and false if was not (because it does not exist).
func (p *PitCsTree) RemoveInterest(pitEntry PitEntry) bool {
	e := pitEntry.(*nameTreePitEntry) // No error check needed because PitCsTree always uses nameTreePitEntry
	for i, entry := range e.node.pitEntries {
		if entry == pitEntry {
			if len(e.node.pitEntries) > 1 {
				e.node.pitEntries[i] = e.node.pitEntries[len(e.node.pitEntries)-1]
			}
			e.node.pitEntries = e.node.pitEntries[:len(e.node.pitEntries)-1]
			if len(e.node.pitEntries) == 0 {
				entry.node.pruneIfEmpty()
			}
			p.nPitEntries--
			delete(p.pitTokenMap, e.token)
			return true
		}
	}
	return false
}

// FindInterestExactMatch returns the PIT entry for an exact match of the
// given interest.
func (p *PitCsTree) FindInterestExactMatch(interest *ndn.Interest) PitEntry {
	node := p.root.findExactMatchEntry(interest.Name())
	if node != nil {
		for _, curEntry := range node.pitEntries {
			if curEntry.CanBePrefix() == interest.CanBePrefix() && curEntry.MustBeFresh() == interest.MustBeFresh() {
				return curEntry
			}
		}
	}
	return nil
}

// FindInterestPrefixMatchByData returns all interests that could be satisfied
// by the given data.
// Example: If we have interests /a and /a/b, a prefix search for data with name /a/b
// will return PitEntries for both /a and /a/b
func (p *PitCsTree) FindInterestPrefixMatchByData(data *ndn.Data, token *uint32) []PitEntry {
	if token != nil {
		if entry, ok := p.pitTokenMap[*token]; ok && entry.Token() == *token {
			return []PitEntry{entry}
		}
		return nil
	}

	return p.findInterestPrefixMatchByName(data.Name())
}

func (p *PitCsTree) findInterestPrefixMatchByName(name *ndn.Name) []PitEntry {
	matching := make([]PitEntry, 0)
	dataNameLen := name.Size()
	for curNode := p.root.findLongestPrefixEntry(name); curNode != nil; curNode = curNode.parent {
		for _, entry := range curNode.pitEntries {
			if entry.canBePrefix || curNode.depth == dataNameLen {
				matching = append(matching, entry)
			}
		}
	}
	return matching
}

// PitSize returns the number of entries in the PIT.
func (p *PitCsTree) PitSize() int {
	return p.nPitEntries
}

// CsSize returns the number of entries in the CS.
func (p *PitCsTree) CsSize() int {
	return p.nCsEntries
}

// IsCsAdmitting returns whether the CS is admitting content.
func (p *PitCsTree) IsCsAdmitting() bool {
	return csAdmit
}

// IsCsServing returns whether the CS is serving content.
func (p *PitCsTree) IsCsServing() bool {
	return csServe
}

// InsertOutRecord inserts an outrecord for the given interest, updating the
// preexisting one if it already occcurs.
func (e *nameTreePitEntry) InsertOutRecord(interest *ndn.Interest, face uint64) *PitOutRecord {
	var record *PitOutRecord
	var ok bool
	if record, ok = e.outRecords[face]; !ok {
		record := new(PitOutRecord)
		record.Face = face
		record.LatestNonce = interest.Nonce()
		record.LatestTimestamp = time.Now()
		record.LatestInterest = interest
		record.ExpirationTime = time.Now().Add(interest.Lifetime())
		e.outRecords[face] = record
		return record
	}

	// Existing record
	record.LatestNonce = interest.Nonce()
	record.LatestTimestamp = time.Now()
	record.LatestInterest = interest
	record.ExpirationTime = time.Now().Add(interest.Lifetime())
	return record
}

// GetOutRecords returns all outrecords for the given PIT entry.
func (e *nameTreePitEntry) GetOutRecords() []*PitOutRecord {
	records := make([]*PitOutRecord, 0)
	for _, value := range e.outRecords {
		records = append(records, value)
	}
	return records
}

func (p *pitCsTreeNode) findExactMatchEntry(name *ndn.Name) *pitCsTreeNode {
	if name.Size() > p.depth {
		if child, ok := p.children[name.At(p.depth).String()]; ok {
			return child.findExactMatchEntry(name)
		}
	} else if name.Size() == p.depth {
		return p
	}
	return nil
}

func (p *pitCsTreeNode) findLongestPrefixEntry(name *ndn.Name) *pitCsTreeNode {
	if name.Size() > p.depth {
		if child, ok := p.children[name.At(p.depth).String()]; ok {
			return child.findLongestPrefixEntry(name)
		}
	}
	return p
}

func (p *pitCsTreeNode) fillTreeToPrefix(name *ndn.Name) *pitCsTreeNode {
	curNode := p.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(pitCsTreeNode)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		newNode.children = make(map[string]*pitCsTreeNode)

		curNode.children[newNode.component.String()] = newNode
		curNode = newNode
	}
	return curNode
}

func (p *pitCsTreeNode) getChildrenCount() int {
	return len(p.children)
}

func (p *pitCsTreeNode) pruneIfEmpty() {
	for curNode := p; curNode.parent != nil && curNode.getChildrenCount() == 0 && len(curNode.pitEntries) == 0 && curNode.csEntry == nil; curNode = curNode.parent {
		delete(curNode.parent.children, curNode.component.String())
	}
}

func (p *PitCsTree) generateNewPitToken() uint32 {
	for {
		token := rand.Uint32()
		if _, ok := p.pitTokenMap[token]; !ok {
			return token
		}
	}
}

func (p *PitCsTree) hashCsName(name *ndn.Name) uint64 {
	return xxhash.Sum64String(name.String())
}

// FindMatchingDataFromCS finds the best matching entry in the CS (if any).
// If MustBeFresh is set to true in the Interest, only non-stale CS entries
// will be returned.
func (p *PitCsTree) FindMatchingDataFromCS(interest *ndn.Interest) CsEntry {
	node := p.root.findExactMatchEntry(interest.Name())
	if node != nil {
		if !interest.CanBePrefix() {
			if node.csEntry != nil {
				p.csReplacement.BeforeUse(node.csEntry.index, node.csEntry.data)
				return node.csEntry
			}
			// Return nil instead of node.csEntry so that
			// the return type is nil rather than CSEntry{nil}
			return nil
		}
		return node.findMatchingDataCSPrefix(interest)
	}
	return nil
}

// InsertData inserts a Data packet into the Content Store.
func (p *PitCsTree) InsertData(data *ndn.Data) {
	index := p.hashCsName(data.Name())

	if entry, ok := p.csMap[index]; ok {
		// Replace existing entry
		entry.data = data

		if data.MetaInfo() == nil || data.MetaInfo().FinalBlockID() == nil {
			entry.staleTime = time.Now()
		} else {
			entry.staleTime = time.Now().Add(*data.MetaInfo().FreshnessPeriod())
		}

		p.csReplacement.AfterRefresh(index, data)
	} else {
		// New entry
		p.nCsEntries++
		node := p.root.fillTreeToPrefix(data.Name())
		node.csEntry = new(nameTreeCsEntry)
		node.csEntry.node = node
		node.csEntry.index = index
		node.csEntry.data = data
		p.csMap[index] = node.csEntry
		p.csReplacement.AfterInsert(index, data)

		// Tell replacement strategy to evict entries if needed
		p.csReplacement.EvictEntries()
	}
}

// eraseCsDataFromReplacementStrategy allows the replacement strategy to
// erase the data with the specified name from the Content Store.
func (p *PitCsTree) eraseCsDataFromReplacementStrategy(index uint64) {
	if entry, ok := p.csMap[index]; ok {
		entry.node.csEntry = nil
		delete(p.csMap, index)
		p.nCsEntries--
	}
}

// Given a pitCsTreeNode that is the longest prefix match of an interest, look for any
// CS data rechable from this pitCsTreeNode. This function must be called only after
// the interest as far as possible with the nodes components in the PitCSTree.
// For example, if we have data for /a/b/v=10 and the interest is /a/b,
// p should be the `b` node, not the root node.
func (p *pitCsTreeNode) findMatchingDataCSPrefix(interest *ndn.Interest) CsEntry {
	if p.csEntry != nil && (!interest.MustBeFresh() || time.Now().Before(p.csEntry.staleTime)) {
		// A csEntry exists at this node and is acceptable to satisfy the interest
		return p.csEntry
	}

	// No csEntry at current node, look farther down the tree
	// We must have already matched the entire interest name
	if p.depth >= interest.Name().Size() {
		for _, child := range p.children {
			potentialMatch := child.findMatchingDataCSPrefix(interest)
			if potentialMatch != nil {
				return potentialMatch
			}
		}
	}

	// If found none, then return
	return nil
}
