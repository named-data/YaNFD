/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/dispatch"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/table"
)

// MaxFwThreads Maximum number of forwarding threads
const MaxFwThreads = 32

// Threads contains all forwarding threads
var Threads map[int]*Thread

// HashNameToFwThread hashes an NDN name to a forwarding thread.
func HashNameToFwThread(name *ndn.Name) int {
	// Dispatch all management requests to thread 0
	if name.Size() > 0 && name.At(0).String() == "localhost" {
		return 0
	}

	return int(xxhash.Sum64String(name.String()) % uint64(len(Threads)))
}

// HashNameToAllPrefixFwThreads hahes an NDN name to all forwarding threads for all prefixes of the name.
func HashNameToAllPrefixFwThreads(name *ndn.Name) []int {
	// Dispatch all management requests to thread 0
	if name.Size() > 0 && name.At(0).String() == "localhost" {
		return []int{0}
	}

	threadMap := make(map[int]interface{})

	// Strings are likely better to work with for performance here than calling Name.prefix
	for nameString := name.String(); len(nameString) > 1; nameString = nameString[:strings.LastIndex(nameString, "/")] {
		threadMap[int(xxhash.Sum64String(nameString)%uint64(len(Threads)))] = true
	}

	threadList := make([]int, 0, len(threadMap))
	for i := range threadMap {
		threadList = append(threadList, i)
	}
	return threadList
}

// Thread Represents a forwarding thread
type Thread struct {
	threadID         int
	pendingInterests chan *ndn.PendingPacket
	pendingDatas     chan *ndn.PendingPacket
	pitCS            *table.PitCs
	strategies       map[string]Strategy
	deadNonceList    *table.DeadNonceList
	shouldQuit       chan interface{}
	HasQuit          chan interface{}

	// Counters
	NInInterests          uint64
	NInData               uint64
	NOutInterests         uint64
	NOutData              uint64
	NSatisfiedInterests   uint64
	NUnsatisfiedInterests uint64
}

// NewThread creates a new forwarding thread
func NewThread(id int) *Thread {
	t := new(Thread)
	t.threadID = id
	t.pendingInterests = make(chan *ndn.PendingPacket, fwQueueSize)
	t.pendingDatas = make(chan *ndn.PendingPacket, fwQueueSize)
	t.pitCS = table.NewPitCS()
	t.strategies = InstantiateStrategies(t)
	t.deadNonceList = table.NewDeadNonceList()
	t.shouldQuit = make(chan interface{}, 1)
	t.HasQuit = make(chan interface{})
	return t
}

func (t *Thread) String() string {
	return "FwThread-" + strconv.Itoa(t.threadID)
}

// GetID returns the ID of the forwarding thread
func (t *Thread) GetID() int {
	return t.threadID
}

// GetNumPitEntries returns the number of entries in this thread's PIT.
func (t *Thread) GetNumPitEntries() int {
	return t.pitCS.PitSize()
}

// GetNumCsEntries returns the number of entries in this thread's ContentStore.
func (t *Thread) GetNumCsEntries() int {
	return t.pitCS.CsSize()
}

// TellToQuit tells the forwarding thread to quit
func (t *Thread) TellToQuit() {
	core.LogInfo(t, "Told to quit")
	t.shouldQuit <- true
}

// Run forwarding thread
func (t *Thread) Run() {
	for !core.ShouldQuit {
		select {
		case pendingPacket := <-t.pendingInterests:
			t.processIncomingInterest(pendingPacket)
		case pendingPacket := <-t.pendingDatas:
			t.processIncomingData(pendingPacket)
		case expiringPitEntry := <-t.pitCS.ExpiringPitEntries:
			t.finalizeInterest(expiringPitEntry)
		case <-t.deadNonceList.ExpirationTimer:
			t.deadNonceList.RemoveExpiredEntry()
		case <-t.shouldQuit:
			continue
		}
	}

	core.LogInfo(t, "Stopping thread")
	t.HasQuit <- true
}

// QueueInterest queues an Interest for processing by this forwarding thread.
func (t *Thread) QueueInterest(interest *ndn.PendingPacket) {
	t.pendingInterests <- interest
}

// QueueData queues a DAta packet for processing by this forwarding thread.
func (t *Thread) QueueData(data *ndn.PendingPacket) {
	t.pendingDatas <- data
}

func (t *Thread) processIncomingInterest(pendingPacket *ndn.PendingPacket) {
	// Ensure incoming face is indicated
	if pendingPacket.IncomingFaceID == nil {
		core.LogError(t, "Interest missing IncomingFaceId - DROP")
		return
	}

	// Extract Interest from PendingPacket
	interest, err := ndn.DecodeInterest(pendingPacket.Wire)
	if err != nil {
		core.LogInfo(t, "Unable to decode Interest packet - DROP")
		return
	}

	// Get incoming face
	incomingFace := dispatch.GetFace(*pendingPacket.IncomingFaceID)
	if incomingFace == nil {
		core.LogError(t, "Non-existent incoming FaceID="+strconv.Itoa(int(*pendingPacket.IncomingFaceID))+" for Interest="+interest.Name().String()+" - DROP")
		return
	}

	// Drop if HopLimit present and is 0. Else, decrement by 1
	if interest.HopLimit() != nil && *interest.HopLimit() == 0 {
		core.LogDebug(t, "Received Interest="+interest.Name().String()+" with HopLimit=0 - DROP")
		return
	} else if interest.HopLimit() != nil {
		interest.SetHopLimit(*interest.HopLimit() - 1)
	}

	// Get PIT token (if any)
	incomingPitToken := make([]byte, 0)
	if len(pendingPacket.PitToken) > 0 {
		incomingPitToken = make([]byte, len(pendingPacket.PitToken))
		copy(incomingPitToken, pendingPacket.PitToken)
		core.LogTrace(t, "OnIncomingInterest: "+interest.Name().String()+", FaceID="+strconv.FormatUint(incomingFace.FaceID(), 10)+", Has PitToken")
	} else {
		core.LogTrace(t, "OnIncomingInterest: "+interest.Name().String()+", FaceID="+strconv.FormatUint(incomingFace.FaceID(), 10))
	}

	// Check if violates /localhost
	if incomingFace.Scope() == ndn.NonLocal && interest.Name().Size() > 0 && interest.Name().At(0).String() == "localhost" {
		core.LogWarn(t, "Interest "+interest.Name().String()+" from non-local face="+strconv.FormatUint(incomingFace.FaceID(), 10)+" violates /localhost scope - DROP")
		return
	}

	t.NInInterests++

	// Detect duplicate nonce by comparing against Dead Nonce List
	if _, exists := t.deadNonceList.Find(interest.Name(), interest.Nonce()); exists {
		core.LogTrace(t, "Interest "+interest.Name().String()+" matches Dead Nonce List - DROP")
		return
	}

	// Check for forwarding hint and, if present, determine if reaching producer region (and then strip forwarding hint)
	isReachingProducerRegion := true
	var forwardingHint *ndn.Delegation
	if len(interest.ForwardingHint()) > 0 {
		isReachingProducerRegion = false
		for _, fh := range interest.ForwardingHint() {
			if table.NetworkRegion.IsProducer(fh.Name()) {
				isReachingProducerRegion = true
				break
			} else if forwardingHint == nil || fh.Preference() < forwardingHint.Preference() {
				forwardingHint = &fh
			}
		}

		if isReachingProducerRegion {
			interest.ClearForwardingHints()
			forwardingHint = nil
		}
	}

	// Check if any matching PIT entries (and if duplicate)
	pitEntry, isDuplicate := t.pitCS.FindOrInsertPIT(interest, forwardingHint, incomingFace.FaceID())
	if isDuplicate {
		// Interest loop - since we don't use Nacks, just drop
		core.LogInfo(t, "Interest "+interest.Name().String()+" is looping - DROP")
		return
	}
	core.LogDebug(t, "Found or updated PIT entry for Interest="+interest.Name().String()+", PitToken="+strconv.FormatUint(uint64(pitEntry.Token), 10))

	// Get strategy for name
	strategyName := table.FibStrategyTable.LongestPrefixStrategy(interest.Name())
	strategy := t.strategies[strategyName.String()]
	core.LogDebug(t, "Using Strategy="+strategyName.String()+" for Interest="+interest.Name().String())

	// Add in-record and determine if already pending
	_, isAlreadyPending := pitEntry.FindOrInsertInRecord(interest, incomingFace.FaceID(), incomingPitToken)
	if !isAlreadyPending {
		core.LogTrace(t, "Interest "+interest.Name().String()+" is not pending")

		// Check CS for matching entry
		if t.pitCS.IsCsServing() {
			csEntry := t.pitCS.FindMatchingDataCS(interest)
			if csEntry != nil {
				// Pass to strategy AfterContentStoreHit pipeline
				strategy.AfterContentStoreHit(pitEntry, incomingFace.FaceID(), csEntry.Data)
				return
			}
		}
	} else {
		core.LogTrace(t, "Interest "+interest.Name().String()+" is already pending")
	}

	// Update PIT entry expiration timer
	pitEntry.UpdateExpirationTimer()

	// If NextHopFaceId set, forward to that face (if it exists) or drop
	if pendingPacket.NextHopFaceID != nil {
		if dispatch.GetFace(*pendingPacket.NextHopFaceID) != nil {
			core.LogTrace(t, "NextHopFaceId is set for Interest "+interest.Name().String()+" - dispatching directly to face")
			dispatch.GetFace(*pendingPacket.NextHopFaceID).SendPacket(pendingPacket)
		} else {
			core.LogInfo(t, "Non-existent face specified in NextHopFaceId for Interest "+interest.Name().String()+" - DROP")
		}
		return
	}

	// Pass to strategy AfterReceiveInterest pipeline
	var nexthops []*table.FibNextHopEntry
	if forwardingHint == nil {
		nexthops = table.FibStrategyTable.LongestPrefixNexthops(interest.Name())
	} else {
		nexthops = table.FibStrategyTable.LongestPrefixNexthops(forwardingHint.Name())
	}
	strategy.AfterReceiveInterest(pitEntry, incomingFace.FaceID(), interest, nexthops)
}

func (t *Thread) processOutgoingInterest(interest *ndn.Interest, pitEntry *table.PitEntry, nexthop uint64, inFace uint64) {
	core.LogTrace(t, "OnOutgoingInterest: "+interest.Name().String()+", FaceID="+strconv.FormatUint(nexthop, 10))

	// Get outgoing face
	outgoingFace := dispatch.GetFace(nexthop)
	if outgoingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID="+strconv.FormatUint(nexthop, 10)+" for Interest="+interest.Name().String()+" - DROP")
		return
	}

	// Drop if HopLimit (if present) on Interest going to non-local face is 0. If so, drop
	if interest.HopLimit() != nil && *interest.HopLimit() == 0 && outgoingFace.Scope() == ndn.NonLocal {
		core.LogDebug(t, "Attempting to send Interest="+interest.Name().String()+" with HopLimit=0 to non-local face - DROP")
		return
	}

	// Create or update out-record
	pitEntry.FindOrInsertOutRecord(interest, nexthop)

	t.NOutInterests++

	// Send on outgoing face
	pendingPacket := new(ndn.PendingPacket)
	pendingPacket.IncomingFaceID = new(uint64)
	*pendingPacket.IncomingFaceID = uint64(inFace)
	pendingPacket.PitToken = make([]byte, 6)
	binary.BigEndian.PutUint16(pendingPacket.PitToken, uint16(t.threadID))
	binary.BigEndian.PutUint32(pendingPacket.PitToken[2:], pitEntry.Token)
	var err error
	pendingPacket.Wire, err = interest.Encode()
	if err != nil {
		core.LogWarn(t, "Unable to encode Interest "+interest.Name().String()+" ("+err.Error()+" ) - DROP")
		return
	}
	outgoingFace.SendPacket(pendingPacket)
}

func (t *Thread) finalizeInterest(pitEntry *table.PitEntry) {
	core.LogTrace(t, "OnFinalizeInterest: "+pitEntry.Name.String())

	// Check for nonces to insert into dead nonce list
	for _, outRecord := range pitEntry.OutRecords {
		t.deadNonceList.Insert(outRecord.LatestInterest.Name(), outRecord.LatestNonce)
	}

	// Counters
	if !pitEntry.Satisfied {
		t.NUnsatisfiedInterests += uint64(len(pitEntry.InRecords))
	}

	// Remove from PIT
	t.pitCS.RemovePITEntry(pitEntry)
}

func (t *Thread) processIncomingData(pendingPacket *ndn.PendingPacket) {
	// Ensure incoming face is indicated
	if pendingPacket.IncomingFaceID == nil {
		core.LogError(t, "Data missing IncomingFaceId - DROP")
		return
	}

	// Get PIT if present
	var pitToken *uint32
	if len(pendingPacket.PitToken) > 0 {
		pitToken = new(uint32)
		// We have already guaranteed that, if a PIT token is present, it is 6 bytes long
		*pitToken = binary.BigEndian.Uint32(pendingPacket.PitToken[2:6])
	}

	// Extract Data from PendingPacket
	data, err := ndn.DecodeData(pendingPacket.Wire, false)
	if err != nil {
		core.LogInfo(t, "Unable to decode Data packet - DROP")
		return
	}

	// Get incoming face
	incomingFace := dispatch.GetFace(*pendingPacket.IncomingFaceID)
	if incomingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID="+strconv.Itoa(int(*pendingPacket.IncomingFaceID))+" for Data="+data.Name().String()+" DROP")
		return
	}

	core.LogTrace(t, "OnIncomingData: "+data.Name().String()+", FaceID="+strconv.FormatUint(incomingFace.FaceID(), 10))

	t.NInData++

	// Check if violates /localhost
	if incomingFace.Scope() == ndn.NonLocal && data.Name().Size() > 0 && data.Name().At(0).String() == "localhost" {
		core.LogWarn(t, "Data "+data.Name().String()+" from non-local FaceID="+strconv.FormatUint(*pendingPacket.IncomingFaceID, 10)+" violates /localhost scope - DROP")
		return
	}

	// Add to Content Store
	if t.pitCS.IsCsAdmitting() {
		t.pitCS.InsertDataCS(data)
	}

	// Check for matching PIT entries
	pitEntries := t.pitCS.FindPITFromData(data, pitToken)
	if len(pitEntries) == 0 {
		// Unsolicated Data - nothing more to do
		core.LogDebug(t, "Unsolicited data "+data.Name().String()+" - DROP")
		return
	}

	// Get strategy for name
	strategyName := table.FibStrategyTable.LongestPrefixStrategy(data.Name())
	strategy := t.strategies[strategyName.String()]

	if len(pitEntries) == 1 {
		// Set PIT entry expiration to now
		pitEntries[0].SetExpirationTimerToNow()

		// Invoke strategy's AfterReceiveData
		core.LogTrace(t, "Sending Data="+data.Name().String()+" to strategy="+strategyName.String())
		strategy.AfterReceiveData(pitEntries[0], *pendingPacket.IncomingFaceID, data)

		// Mark PIT entry as satisfied
		pitEntries[0].Satisfied = true

		// Insert into dead nonce list
		for _, outRecord := range pitEntries[0].OutRecords {
			t.deadNonceList.Insert(outRecord.LatestInterest.Name(), outRecord.LatestNonce)
		}

		// Clear out records from PIT entry
		pitEntries[0].ClearOutRecords()
	} else {
		for _, pitEntry := range pitEntries {
			// Store all pending downstreams (except face Data packet arrived on) and PIT tokens
			downstreams := make(map[uint64][]byte)
			for downstreamFaceID, downstreamFaceRecord := range pitEntry.InRecords {
				if downstreamFaceID != *pendingPacket.IncomingFaceID {
					// TODO: Ad-hoc faces
					downstreams[downstreamFaceID] = make([]byte, len(downstreamFaceRecord.PitToken))
					copy(downstreams[downstreamFaceID], downstreamFaceRecord.PitToken)
				}
			}

			// Set PIT entry expiration to now
			pitEntry.SetExpirationTimerToNow()

			// Invoke strategy's BeforeSatisfyInterest
			strategy.BeforeSatisfyInterest(pitEntry, *pendingPacket.IncomingFaceID, data)

			// Mark PIT entry as satisfied
			pitEntry.Satisfied = true

			// Insert into dead nonce list
			for _, outRecord := range pitEntries[0].OutRecords {
				t.deadNonceList.Insert(outRecord.LatestInterest.Name(), outRecord.LatestNonce)
			}

			// Clear PIT entry's in- and out-records
			pitEntry.ClearInRecords()
			pitEntry.ClearOutRecords()

			// Call outoing Data pipeline for each pending downstream
			for downstreamFaceID, downstreamPITToken := range downstreams {
				core.LogTrace(t, "Multiple matching PIT entries for "+data.Name().String()+": sending to do OnOutgoingData pipeline")
				t.processOutgoingData(data, downstreamFaceID, downstreamPITToken, *pendingPacket.IncomingFaceID)
			}
		}
	}
}

func (t *Thread) processOutgoingData(data *ndn.Data, nexthop uint64, pitToken []byte, inFace uint64) {
	core.LogTrace(t, "OnOutgoingData: "+data.Name().String()+", FaceID="+strconv.FormatUint(nexthop, 10))

	// Get outgoing face
	outgoingFace := dispatch.GetFace(nexthop)
	if outgoingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID="+strconv.FormatUint(nexthop, 10)+" for Data="+data.Name().String()+" - DROP")
		return
	}

	// Check if violates /localhost
	if outgoingFace.Scope() == ndn.NonLocal && data.Name().Size() > 0 && data.Name().At(0).String() == "localhost" {
		core.LogWarn(t, "Data "+data.Name().String()+" cannot be sent to non-local FaceID="+strconv.FormatUint(nexthop, 10)+" since violates /localhost scope - DROP")
		return
	}

	t.NOutData++
	t.NSatisfiedInterests++

	// Send on outgoing face
	pendingPacket := new(ndn.PendingPacket)
	var err error
	if len(pitToken) > 0 {
		pendingPacket.PitToken = make([]byte, len(pitToken))
		copy(pendingPacket.PitToken, pitToken)
	}
	pendingPacket.IncomingFaceID = new(uint64)
	*pendingPacket.IncomingFaceID = uint64(inFace)
	pendingPacket.Wire, err = data.Encode()
	if err != nil {
		core.LogWarn(t, "Unable to encode Data "+data.Name().String()+" ("+err.Error()+" ) - DROP")
		return
	}
	outgoingFace.SendPacket(pendingPacket)
}
