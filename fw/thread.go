/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"strconv"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/defn"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// MaxFwThreads Maximum number of forwarding threads
const MaxFwThreads = 32

// Threads contains all forwarding threads
var Threads map[int]*Thread
var LOCALHOST = []byte{0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x68, 0x6f, 0x73, 0x74}

// HashNameToFwThread hashes an NDN name to a forwarding thread.
func HashNameToFwThread(name enc.Name) int {
	// Dispatch all management requests to thread 0
	// this is fine, all it does is make sure the pitcs table in thread 0 has the management stuff.
	// This is not actually touching management.
	if len(name) > 0 && bytes.Equal((name)[0].Val, LOCALHOST) {
		return 0
	}
	// to prevent negative modulos because we converted from uint to int
	return int(name.Hash() % uint64(len(Threads)))
}

// HashNameToAllPrefixFwThreads hashes an NDN name to all forwarding threads for all prefixes of the name.
func HashNameToAllPrefixFwThreads(name enc.Name) []int {
	// Dispatch all management requests to thread 0
	if len(name) > 0 && bytes.Equal((name)[0].Val, LOCALHOST) {
		return []int{0}
	}

	// Strings are likely better to work with for performance here than calling Name.prefix
	// for nameString := (*name); len(nameString) > 1; nameString = nameString[:len(nameString)-1] {
	// 	threadMap[int(xxhash.Sum64(nameString.Bytes())%uint64(len(Threads)))] = true
	// }
	threadList := make([]int, 0, len(Threads))
	prefixHash := name.PrefixHash()
	for i := 1; i < len(prefixHash); i++ {
		h := prefixHash[i]
		threadList = append(threadList, int(h%uint64(len(Threads))))
	}
	return threadList
}

// Thread Represents a forwarding thread
type Thread struct {
	threadID         int
	pendingInterests chan *defn.Pkt
	pendingDatas     chan *defn.Pkt
	pitCS            table.PitCsTable
	strategies       map[uint64]Strategy
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
	t.pendingInterests = make(chan *defn.Pkt, fwQueueSize)
	t.pendingDatas = make(chan *defn.Pkt, fwQueueSize)
	t.pitCS = table.NewPitCS(t.finalizeInterest)
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
	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	pitUpdateTimer := t.pitCS.UpdateTimer()
	for !core.ShouldQuit {
		select {
		case pendingPacket := <-t.pendingInterests:
			t.processIncomingInterest(pendingPacket)
		case pendingPacket := <-t.pendingDatas:
			t.processIncomingData(pendingPacket)
		case <-t.deadNonceList.Ticker.C:
			t.deadNonceList.RemoveExpiredEntries()
		case <-pitUpdateTimer:
			t.pitCS.Update()
		case <-t.shouldQuit:
			continue
		}
	}

	t.deadNonceList.Ticker.Stop()

	core.LogInfo(t, "Stopping thread")
	t.HasQuit <- true
}

// QueueInterest queues an Interest for processing by this forwarding thread.
func (t *Thread) QueueInterest(interest *defn.Pkt) {
	select {
	case t.pendingInterests <- interest:
	default:
		core.LogError(t, "Interest dropped due to full queue")
	}
}

// QueueData queues a Data packet for processing by this forwarding thread.
func (t *Thread) QueueData(data *defn.Pkt) {
	select {
	case t.pendingDatas <- data:
	default:
		core.LogError(t, "Data dropped due to full queue")
	}
}

func (t *Thread) processIncomingInterest(packet *defn.Pkt) {
	interest := packet.L3.Interest
	if interest == nil {
		panic("processIncomingInterest called with non-Interest packet")
	}

	// Ensure incoming face is indicated
	if packet.IncomingFaceID == nil {
		core.LogError(t, "Interest missing IncomingFaceId - DROP")
		return
	}
	// Already asserted that this is an Interest in link service
	// Get incoming face
	incomingFace := dispatch.GetFace(*packet.IncomingFaceID)
	if incomingFace == nil {
		core.LogError(t, "Non-existent incoming FaceID=", *packet.IncomingFaceID,
			" for Interest=", packet.Name, " - DROP")
		return
	}

	if interest.HopLimitV != nil {
		core.LogTrace(t, "Interest ", packet.Name, " has HopLimit=", *interest.HopLimitV)
		if *interest.HopLimitV == 0 {
			return
		}
		*interest.HopLimitV -= 1
	}

	// Log PIT token (if any)
	core.LogTrace(t, "OnIncomingInterest: ", packet.Name, ", FaceID=", incomingFace.FaceID(), ", PitTokenL=", len(packet.PitToken))

	// Check if violates /localhost
	if incomingFace.Scope() == defn.NonLocal &&
		len(interest.NameV) > 0 &&
		bytes.Equal(interest.NameV[0].Val, LOCALHOST) {
		core.LogWarn(t, "Interest ", packet.Name, " from non-local face=", incomingFace.FaceID(), " violates /localhost scope - DROP")
		return
	}

	t.NInInterests++

	// Check for forwarding hint and, if present, determine if reaching producer region (and then strip forwarding hint)
	isReachingProducerRegion := true
	var fhName enc.Name = nil
	hint := interest.ForwardingHintV
	if hint != nil && len(hint.Names) > 0 {
		isReachingProducerRegion = false
		for _, fh := range hint.Names {
			if table.NetworkRegion.IsProducer(fh) {
				isReachingProducerRegion = true
				break
			} else if fhName == nil {
				fhName = fh
			}
		}
		if isReachingProducerRegion {
			//interest.SetForwardingHint(nil)
			//will need to add this back again!
			// TODO: Unable to drop the forwarding hint for now.
			fhName = nil
		}
	}

	// Drop packet if no nonce is found
	if interest.NonceV == nil {
		core.LogDebug(t, "Interest ", packet.Name, " is missing Nonce - DROP")
		return
	}

	// Check if packet is in dead nonce list
	if exists := t.deadNonceList.Find(interest.NameV, *interest.NonceV); exists {
		core.LogDebug(t, "Interest ", packet.Name, " is dropped by DeadNonce: ", *interest.NonceV)
		return
	}

	// Check if any matching PIT entries (and if duplicate)
	//read into this, looks like this one will have to be manually changed
	pitEntry, isDuplicate := t.pitCS.InsertInterest(interest, fhName, incomingFace.FaceID())
	if isDuplicate {
		// Interest loop - since we don't use Nacks, just drop
		core.LogDebug(t, "Interest ", packet.Name, " is looping - DROP")
		return
	}

	// Get strategy for name
	strategyName := table.FibStrategyTable.FindStrategyEnc(interest.NameV)
	strategy := t.strategies[strategyName.Hash()]

	// Add in-record and determine if already pending
	// this looks like custom interest again, but again can be changed without much issue?
	_, isAlreadyPending, prevNonce := pitEntry.InsertInRecord(
		interest, incomingFace.FaceID(), packet.PitToken)

	if !isAlreadyPending {
		core.LogTrace(t, "Interest ", packet.Name, " is not pending")

		// Check CS for matching entry
		if t.pitCS.IsCsServing() {
			csEntry := t.pitCS.FindMatchingDataFromCS(interest)
			if csEntry != nil {
				// Parse the cached data packet and replace in the pending one
				// This is not the fastest way to do it, but simplifies everything
				// significantly. We can optimize this later.
				csData, csWire, err := csEntry.Copy()
				if csData != nil && csWire != nil {
					packet.L3.Data = csData
					packet.L3.Interest = nil
					packet.Raw = csWire
					packet.Name = csData.NameV
					strategy.AfterContentStoreHit(packet, pitEntry, incomingFace.FaceID())
					return
				} else if err != nil {
					core.LogError(t, "Error copying CS entry: ", err)
				} else {
					core.LogError(t, "Error copying CS entry: csData is nil")
				}

			}
		}
	} else {
		core.LogTrace(t, "Interest ", packet.Name, " is already pending")

		// Add the previous nonce to the dead nonce list to prevent further looping
		// TODO: review this design, not specified in NFD dev guide
		t.deadNonceList.Insert(interest.NameV, prevNonce)
	}

	// Update PIT entry expiration timer
	table.UpdateExpirationTimer(pitEntry)

	// If NextHopFaceId set, forward to that face (if it exists) or drop
	if packet.NextHopFaceID != nil {
		if dispatch.GetFace(*packet.NextHopFaceID) != nil {
			core.LogTrace(t, "NextHopFaceId is set for Interest ", packet.Name, " - dispatching directly to face")
			dispatch.GetFace(*packet.NextHopFaceID).SendPacket(dispatch.OutPkt{
				Pkt:      packet,
				PitToken: packet.PitToken, // TODO: ??
				InFace:   packet.IncomingFaceID,
			})
		} else {
			core.LogInfo(t, "Non-existent face specified in NextHopFaceId for Interest ", packet.Name, " - DROP")
		}
		return
	}

	// Use forwarding hint if present
	lookupName := interest.NameV
	if fhName != nil {
		lookupName = fhName
	}

	// Query the FIB for possible nexthops
	nexthops := table.FibStrategyTable.FindNextHopsEnc(lookupName)

	// Exclude faces that have an in-record for this interest
	// TODO: unclear where NFD dev guide specifies such behavior (if any)
	allowedNexthops := make([]*table.FibNextHopEntry, 0, len(nexthops))
	for _, nexthop := range nexthops {
		record := pitEntry.InRecords()[nexthop.Nexthop]
		if record == nil || nexthop.Nexthop == incomingFace.FaceID() {
			allowedNexthops = append(allowedNexthops, nexthop)
		}
	}

	// Pass to strategy AfterReceiveInterest pipeline
	strategy.AfterReceiveInterest(packet, pitEntry, incomingFace.FaceID(), allowedNexthops)
}

func (t *Thread) processOutgoingInterest(
	packet *defn.Pkt,
	pitEntry table.PitEntry,
	nexthop uint64,
	inFace uint64,
) bool {
	interest := packet.L3.Interest
	if interest == nil {
		panic("processOutgoingInterest called with non-Interest packet")
	}

	core.LogTrace(t, "OnOutgoingInterest: ", packet.Name, ", FaceID=", nexthop)

	// Get outgoing face
	outgoingFace := dispatch.GetFace(nexthop)
	if outgoingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID=", nexthop, " for Interest=", packet.Name, " - DROP")
		return false
	}
	if outgoingFace.FaceID() == inFace && outgoingFace.LinkType() != defn.AdHoc {
		core.LogDebug(t, "Attempting to send Interest=", packet.Name, " back to incoming face - DROP")
		return false
	}

	// Drop if HopLimit (if present) on Interest going to non-local face is 0. If so, drop
	if interest.HopLimitV != nil && int(*interest.HopLimitV) == 0 &&
		outgoingFace.Scope() == defn.NonLocal {
		core.LogDebug(t, "Attempting to send Interest=", packet.Name, " with HopLimit=0 to non-local face - DROP")
		return false
	}

	// Create or update out-record
	pitEntry.InsertOutRecord(interest, nexthop)

	t.NOutInterests++

	// Make new PIT token if needed
	pitToken := make([]byte, 6)
	binary.BigEndian.PutUint16(pitToken, uint16(t.threadID))
	binary.BigEndian.PutUint32(pitToken[2:], pitEntry.Token())

	// Send on outgoing face
	outgoingFace.SendPacket(dispatch.OutPkt{
		Pkt:      packet,
		PitToken: pitToken,
		InFace:   utils.IdPtr(inFace),
	})

	return true
}

func (t *Thread) finalizeInterest(pitEntry table.PitEntry) {
	// Check for nonces to insert into dead nonce list
	for _, outRecord := range pitEntry.OutRecords() {
		t.deadNonceList.Insert(outRecord.LatestInterest, outRecord.LatestNonce)
	}

	// Counters
	if !pitEntry.Satisfied() {
		t.NUnsatisfiedInterests += uint64(len(pitEntry.InRecords()))
	}
}

func (t *Thread) processIncomingData(packet *defn.Pkt) {
	data := packet.L3.Data
	if data == nil {
		panic("processIncomingData called with non-Data packet")
	}

	// Ensure incoming face is indicated
	if packet.IncomingFaceID == nil {
		core.LogError(t, "Data missing IncomingFaceId - DROP")
		return
	}

	// Get PIT if present
	var pitToken *uint32
	//lint:ignore S1009 removing the nil check causes a segfault ¯\_(ツ)_/¯
	if packet.PitToken != nil && len(packet.PitToken) == 6 {
		pitToken = utils.IdPtr(binary.BigEndian.Uint32(packet.PitToken[2:6]))
	}

	// Get incoming face
	incomingFace := dispatch.GetFace(*packet.IncomingFaceID)
	if incomingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID=", *packet.IncomingFaceID, " for Data=", packet.Name, " DROP")
		return
	}

	t.NInData++

	// Check if violates /localhost
	if incomingFace.Scope() == defn.NonLocal && len(packet.Name) > 0 &&
		bytes.Equal(data.NameV[0].Val, LOCALHOST) {
		core.LogWarn(t, "Data ", packet.Name, " from non-local FaceID=", *packet.IncomingFaceID, " violates /localhost scope - DROP")
		return
	}

	// Add to Content Store
	if t.pitCS.IsCsAdmitting() {
		t.pitCS.InsertData(data, packet.Raw)
	}

	// Check for matching PIT entries
	pitEntries := t.pitCS.FindInterestPrefixMatchByDataEnc(data, pitToken)
	if len(pitEntries) == 0 {
		// Unsolicated Data - nothing more to do
		core.LogDebug(t, "Unsolicited data ", packet.Name, " FaceID=", *packet.IncomingFaceID, " - DROP")
		return
	}

	// Get strategy for name
	strategyName := table.FibStrategyTable.FindStrategyEnc(data.NameV)
	strategy := t.strategies[strategyName.Hash()]

	if len(pitEntries) == 1 {
		// Set PIT entry expiration to now
		table.SetExpirationTimerToNow(pitEntries[0])

		// Invoke strategy's AfterReceiveData
		core.LogTrace(t, "Sending Data=", packet.Name, " to strategy=", strategyName)
		strategy.AfterReceiveData(packet, pitEntries[0], *packet.IncomingFaceID)

		// Mark PIT entry as satisfied
		pitEntries[0].SetSatisfied(true)

		// Insert into dead nonce list
		for _, outRecord := range pitEntries[0].OutRecords() {
			t.deadNonceList.Insert(data.NameV, outRecord.LatestNonce)
		}

		// Clear out records from PIT entry
		pitEntries[0].ClearInRecords()
		pitEntries[0].ClearOutRecords()
	} else {
		for _, pitEntry := range pitEntries {
			// Store all pending downstreams (except face Data packet arrived on) and PIT tokens
			downstreams := make(map[uint64][]byte)
			for downstreamFaceID, downstreamFaceRecord := range pitEntry.InRecords() {
				if downstreamFaceID != *packet.IncomingFaceID {
					// TODO: Ad-hoc faces
					downstreams[downstreamFaceID] = make([]byte, len(downstreamFaceRecord.PitToken))
					copy(downstreams[downstreamFaceID], downstreamFaceRecord.PitToken)
				}
			}

			// Set PIT entry expiration to now
			table.SetExpirationTimerToNow(pitEntry)

			// Invoke strategy's BeforeSatisfyInterest
			strategy.BeforeSatisfyInterest(pitEntry, *packet.IncomingFaceID)

			// Mark PIT entry as satisfied
			pitEntry.SetSatisfied(true)

			// Insert into dead nonce list
			for _, outRecord := range pitEntries[0].GetOutRecords() {
				t.deadNonceList.Insert(data.NameV, outRecord.LatestNonce)
			}

			// Clear PIT entry's in- and out-records
			pitEntry.ClearInRecords()
			pitEntry.ClearOutRecords()

			// Call outoing Data pipeline for each pending downstream
			for downstreamFaceID, downstreamPITToken := range downstreams {
				core.LogTrace(t, "Multiple matching PIT entries for ", packet.Name, ": sending to OnOutgoingData pipeline")
				t.processOutgoingData(packet, downstreamFaceID, downstreamPITToken, *packet.IncomingFaceID)
			}
		}
	}
}

func (t *Thread) processOutgoingData(
	packet *defn.Pkt,
	nexthop uint64,
	pitToken []byte,
	inFace uint64,
) {
	data := packet.L3.Data
	if data == nil {
		panic("processOutgoingData called with non-Data packet")
	}

	core.LogTrace(t, "OnOutgoingData: ", packet.Name, ", FaceID=", nexthop)

	// Get outgoing face
	outgoingFace := dispatch.GetFace(nexthop)
	if outgoingFace == nil {
		core.LogError(t, "Non-existent nexthop FaceID=", nexthop, " for Data=", packet.Name, " - DROP")
		return
	}

	// Check if violates /localhost
	if outgoingFace.Scope() == defn.NonLocal && len(data.NameV) > 0 && bytes.Equal(data.NameV[0].Val, LOCALHOST) {
		core.LogWarn(t, "Data ", packet.Name, " cannot be sent to non-local FaceID=", nexthop, " since violates /localhost scope - DROP")
		return
	}

	t.NOutData++
	t.NSatisfiedInterests++

	// Send on outgoing face
	outgoingFace.SendPacket(dispatch.OutPkt{
		Pkt:      packet,
		PitToken: pitToken,
		InFace:   utils.IdPtr(inFace),
	})
}
