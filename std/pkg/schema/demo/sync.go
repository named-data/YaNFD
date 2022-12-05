package demo

import (
	"bytes"
	"errors"
	"math/rand"
	"sync"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
)

type SyncState int

type MissingData struct {
	NodeId   []byte
	StartSeq uint64
	EndSeq   uint64
}

const (
	SyncSteady SyncState = iota
	SyncSupression
)

type SvsOnMissingEvent = func([]byte, uint64, uint64)

// SvsNode represents a subtree supports a simplified state-vector-sync protocol.
// TODO: How to return the missing data to the user? Channel or callback?
// TODO: How can the user express the trust schema here? The `notif` node mat have different requirements
// as the `leaf` node. (#BLACKBOX)
type SvsNode struct {
	schema.BaseNode

	notif *schema.ExpressPoint
	leaf  *schema.LeafNode

	localSv StateVec
	// calledSv StateVec
	state SyncState
	// onMiss       *schema.Event[*SvsOnMissingEvent]
	// sigChan      chan struct{}
	// quitChan     chan struct{}
	missChan     chan MissingData
	syncIntv     time.Duration
	baseMatching enc.Matching

	timer           ndn.Timer
	cancelSyncTimer func() error
	dataLock        sync.Mutex

	selfNodeId  []byte
	channelSize int
	selfSeq     uint64
}

func (n *SvsNode) Init(parent schema.NTNode, edge enc.ComponentPattern) {
	n.BaseNode.Init(parent, edge)
	schema.AddEventListener(n, schema.PropOnAttach, n.onAttach)
	schema.AddEventListener(n, schema.PropOnDetach, n.onDetach)

	// Namespace:
	// - [/prefix]/32=notif -> Notification Interest
	// - [/prefix]/<8=nodeId>/<seq=seqNo>

	pat, _ := enc.NamePatternFromStr("/<8=nodeId>/<seq=seqNo>")
	n.leaf = &schema.LeafNode{}
	n.PutNode(pat, n.leaf)
	// TODO: Discuss if this is a good design. This will be overwritten by the policy setting in onAttach.
	n.leaf.Set(schema.PropCanBePrefix, false)
	n.leaf.Set(schema.PropMustBeFresh, false)
	n.leaf.Set(schema.PropLifetime, 4*time.Second)
	n.leaf.Set(schema.PropFreshness, 60*time.Second)
	n.leaf.Set(schema.PropValidDuration, 876000*time.Hour)

	pat, _ = enc.NamePatternFromStr("/32=notif")
	n.notif = &schema.ExpressPoint{}
	n.PutNode(pat, n.notif)
	n.notif.Set(schema.PropCanBePrefix, true)
	n.notif.Set(schema.PropMustBeFresh, true)
	n.notif.Set(schema.PropLifetime, 1*time.Second)
	schema.AddEventListener(n.notif, schema.PropOnInterest, n.onSyncInt)

	n.baseMatching = enc.Matching{}
	n.channelSize = 1000
}

func (v *StateVec) findSvsEntry(nodeId []byte) int {
	// This is less efficient but enough for a demo.
	for i, n := range v.Entries {
		if bytes.Equal(n.NodeId, nodeId) {
			return i
		}
	}
	return -1
}

func (n *SvsNode) onSyncInt(
	matching enc.Matching, appParam enc.Wire, reply ndn.ReplyFunc, context schema.Context,
) bool {
	remoteSv, err := ParseStateVec(enc.NewWireReader(appParam), true)
	if err != nil {
		name := context[schema.CkName].(enc.Name) // Always succeed
		n.Log.WithField("name", name.String()).Error("Unable to parse state vector. Drop.")
	}

	// If append() is called on localSv slice, a lock is necessary
	n.dataLock.Lock()
	defer n.dataLock.Unlock()

	// Compare state vectors
	// needFetch := false
	needNotif := false
	for _, cur := range remoteSv.Entries {
		li := n.localSv.findSvsEntry(cur.NodeId)
		if li == -1 {
			n.localSv.Entries = append(n.localSv.Entries, &StateVecEntry{
				NodeId: cur.NodeId,
				SeqNo:  cur.SeqNo,
			})
			// needFetch = true
			n.missChan <- MissingData{
				NodeId:   cur.NodeId,
				StartSeq: 1,
				EndSeq:   cur.SeqNo + 1,
			}
		} else if n.localSv.Entries[li].SeqNo < cur.SeqNo {
			n.missChan <- MissingData{
				NodeId:   cur.NodeId,
				StartSeq: n.localSv.Entries[li].SeqNo + 1,
				EndSeq:   cur.SeqNo + 1,
			}
			n.localSv.Entries[li].SeqNo = cur.SeqNo
			// needFetch = true
		} else if n.localSv.Entries[li].SeqNo > cur.SeqNo {
			needNotif = true
		}
	}
	// Notify the callback coroutine if applicable
	// if needFetch {
	// 	select {
	// 	case n.sigChan <- struct{}{}:
	// 	default:
	// 	}
	// }
	// Set sync state if applicable
	// if needNotif {
	// 	n.aggregate(remoteSv)
	// 	if n.state == SyncSteady {
	// 		n.transitToSuppress(remoteSv)
	// 	}
	// }
	// TODO: Have trouble understanding this mechanism from the Spec.
	// From StateVectorSync Spec 4.4,
	// "Incoming Sync Interest is outdated: Node moves to Suppression State."
	// implies the state becomes Supression State when `remote any< local`
	// From StateVectorSync Spec 6, the box below
	// "local_state_vector any< x"
	// implies the state becomes Supression State when `local any< remote`
	// Contradiction. The wrong one should be the figure.
	// Since supression is an optimization that does not affect the demo, ignore for now.
	// Report this issue to the team when have time.

	// Reset the sync timer (already in lock)
	n.cancelSyncTimer()
	if needNotif {
		n.expressStateVec()
	}
	n.cancelSyncTimer = n.timer.Schedule(n.getSyncIntv(), n.onSyncTimer)

	return true
}

// func (n *SvsNode) aggregate(remoteSv *StateVec) {
// 	for _, cur := range remoteSv.Entries {
// 		li := n.aggSv.findSvsEntry(cur.NodeId)
// 		if li == -1 {
// 			n.aggSv.Entries = append(n.aggSv.Entries, &StateVecEntry{
// 				NodeId: cur.NodeId,
// 				SeqNo:  cur.SeqNo,
// 			})
// 		} else {
// 			n.aggSv.Entries[li].SeqNo = utils.Max(n.aggSv.Entries[li].SeqNo, cur.SeqNo)
// 		}
// 	}
// }

// func (n *SvsNode) transitToSuppress(remoteSv *StateVec) {
// 	n.state = SyncSupression
// 	// Set aggregation state vector
// 	n.aggSv = *remoteSv
// 	// Reset Timers

// }

// func (n *SvsNode) onSupressionTimer() {

// }

func (n *SvsNode) onSyncTimer() {
	n.dataLock.Lock()
	defer n.dataLock.Unlock()
	n.expressStateVec()
	// In case a new one is just scheduled by the onInterest callback. No-op most of the case.
	n.cancelSyncTimer()
	n.cancelSyncTimer = n.timer.Schedule(n.getSyncIntv(), n.onSyncTimer)
}

func (n *SvsNode) expressStateVec() {
	n.notif.Need(n.baseMatching, nil, n.localSv.Encode(), schema.Context{})
}

// func (n *SvsNode) callbackRoutine() {
// 	for running := true; running; {
// 		select {
// 		case <-n.quitChan:
// 			running = false
// 			continue
// 		case <-n.sigChan:
// 			for hasNew := false; hasNew; {
// 				for _, cur := range n.localSv.Entries {
// 					ci := n.calledSv.findSvsEntry(cur.NodeId)
// 					if ci == -1 {
// 						n.calledSv.Entries = append(n.calledSv.Entries, &StateVecEntry{
// 							NodeId: cur.NodeId,
// 							SeqNo:  0,
// 						})
// 						ci = len(n.calledSv.Entries) - 1
// 					}
// 					newSeq := cur.SeqNo // Prevent race
// 					if n.calledSv.Entries[ci].SeqNo < newSeq {
// 						// Here, cur.NodeId is never modified; calledSv is only modifiable by this routine.
// 						hasNew = true
// 						for _, evt := range n.onMiss.Val() {
// 							(*evt)(cur.NodeId, n.calledSv.Entries[ci].SeqNo, newSeq)
// 						}
// 						n.calledSv.Entries[ci].SeqNo = newSeq
// 					}
// 				} // loop enum sv
// 			} // loop for updated state
// 		}
// 	}
// }

func (n *SvsNode) getSyncIntv() time.Duration {
	dev := rand.Int63n(n.syncIntv.Nanoseconds()/4) - n.syncIntv.Nanoseconds()/8
	return n.syncIntv + time.Duration(dev)*time.Nanosecond
}

func (n *SvsNode) MissingDataChannel() chan MissingData {
	return n.missChan
}

func (n *SvsNode) MySequence() uint64 {
	return n.selfSeq
}

func (n *SvsNode) NewData(content enc.Wire, context schema.Context) enc.Wire {
	n.dataLock.Lock()
	defer n.dataLock.Unlock()

	n.selfSeq++
	mat := enc.Matching{}
	for k, v := range n.baseMatching {
		mat[k] = v
	}
	mat["nodeId"] = n.selfNodeId
	mat["seqNo"] = uint64(n.selfSeq)
	ret := n.leaf.Provide(mat, nil, content, context)
	if len(ret) > 0 {
		li := n.localSv.findSvsEntry(n.selfNodeId)
		if li >= 0 {
			n.localSv.Entries[li].SeqNo = n.selfSeq
		}
		n.expressStateVec()
	} else {
		n.selfSeq--
	}
	return ret
}

func (n *SvsNode) onAttach(path enc.NamePattern, engine ndn.Engine) error {
	if n.channelSize == 0 || len(n.selfNodeId) == 0 || n.baseMatching == nil || n.syncIntv <= 0 {
		return errors.New("SvsNode: not configured before Init")
	}

	n.timer = engine.Timer()
	n.dataLock = sync.Mutex{}
	n.dataLock.Lock()
	defer n.dataLock.Unlock()

	n.localSv = StateVec{Entries: make([]*StateVecEntry, 0)}
	// n.onMiss = schema.NewEvent[*SvsOnMissingEvent]()
	n.state = SyncSteady
	n.missChan = make(chan MissingData, n.channelSize)
	n.cancelSyncTimer = n.timer.Schedule(n.getSyncIntv(), n.onSyncTimer)
	// go n.callbackRoutine()

	// initialize localSv
	// TODO: this demo does not consider recovery from off-line. Should be done via ENV and storage policy.
	n.localSv.Entries = append(n.localSv.Entries, &StateVecEntry{
		NodeId: n.selfNodeId,
		SeqNo:  0,
	})
	n.selfSeq = 0
	return nil
}

func (n *SvsNode) onDetach(engine ndn.Engine) {
	n.dataLock.Lock()
	defer n.dataLock.Unlock()
	n.cancelSyncTimer()
	close(n.missChan)
}

// Get a property or callback event
func (n *SvsNode) Get(propName schema.PropKey) any {
	if ret := n.BaseNode.Get(propName); ret != nil {
		return ret
	}
	switch propName {
	case "SelfNodeId":
		return n.selfNodeId
	case "ChannelSize":
		return n.channelSize
	case "BaseMatching":
		return n.baseMatching
	case "SyncInterval":
		return n.syncIntv
	}
	return nil
}

// Set a property. Use Get() to update callback events.
func (n *SvsNode) Set(propName schema.PropKey, value any) error {
	if ret := n.BaseNode.Set(propName, value); ret == nil {
		return ret
	}
	switch propName {
	case "SelfNodeId":
		return schema.PropertySet(&n.selfNodeId, propName, value)
	case "ChannelSize":
		return schema.PropertySet(&n.channelSize, propName, value)
	case "BaseMatching":
		return schema.PropertySet(&n.baseMatching, propName, value)
	case "SyncInterval":
		return schema.PropertySet(&n.syncIntv, propName, value)
	}
	return ndn.ErrNotSupported{Item: string(propName)}
}

func (n *SvsNode) Need(
	nodeId []byte, seq uint64, matching enc.Matching, context schema.Context,
) chan schema.NeedResult {
	matching["nodeId"] = nodeId
	matching["seqNo"] = seq
	return n.leaf.Need(matching, nil, nil, context)
}
