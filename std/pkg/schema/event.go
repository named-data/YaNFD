package schema

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

// Event represents the context of a triggered event
type Event struct {
	// TargetNode is the node that triggers the event, valid for all events.
	// Generally equals to Target.Node. Exception: on the callback of Need,
	// Target.Node is matched with Data name but TargetNode is the Interest node.
	TargetNode *Node
	// Target is the target (matched node) that triggers the event
	// Valid for events handling a specific packet.
	Target *MatchedNode
	// Deadline is now + InterestLifetime
	Deadline *time.Time
	// RawPacket is the raw Interest or Data wire
	RawPacket enc.Wire
	// SigCovered contains the bytes of the raw packet that are covered by the signature
	SigCovered enc.Wire
	// Signature is the signature of the packet received.
	Signature ndn.Signature
	// Interest is the received Interest.
	Interest ndn.Interest
	// Data is the received Data.
	Data ndn.Data
	// IntConfig is the config of the Interest that is going to encode.
	IntConfig *ndn.InterestConfig
	// DataConfig is the config of the Data that is going to produce.
	DataConfig *ndn.DataConfig
	// Content is the content of Data or AppParam of Interest.
	Content enc.Wire
	// NackReason is the reason of Network NACK given by the forwarder.
	NackReason *uint64
	// ValidResult is the validation result given by last validator.
	ValidResult *ValidRes
	// SelfProduced indicates whether the Data is produced by myself (this program).
	SelfProduced *bool
	// ValidDuration is The validity period of a data in the storage produced by this node
	// i.e. how long the local storage will serve it.
	// Should be larger than FreshnessPeriod. Not affected data fetched remotely.
	ValidDuration *time.Duration
	// Reply is the func called to reply to an Interest
	Reply ndn.WireReplyFunc
	// NeedStatus is the result status in the callback of need()
	NeedStatus *ndn.InterestResult
	// Error is the optional error happened in an event
	Error error
	// Extra arguments used by application
	Extra map[string]any
}

// Callback represents a callback that handles an event
type Callback = func(event *Event) any

// Event is a chain of callback functions for an event.
// The execution order is supposed to be the addition order.
type EventTarget struct {
	val []*Callback
}

// Add a callback. Note that callback should be a *func.
func (e *EventTarget) Add(callback *Callback) {
	e.val = append(e.val, callback)
}

// Remove a callback
// Seems not useful at all. Do we remove it?
func (e *EventTarget) Remove(callback *Callback) {
	newVal := make([]*Callback, 0, len(e.val))
	for _, v := range e.val {
		if v != callback {
			newVal = append(newVal, v)
		}
	}
	e.val = newVal
}

// Val returns the value of the event. Used by nodes only.
func (e *EventTarget) Val() []*Callback {
	return e.val
}

// Dispatch fires the event, calls every callback and returns the last result
func (e *EventTarget) Dispatch(event *Event) any {
	var ret any = nil
	for _, v := range e.val {
		ret = (*v)(event)
	}
	return ret
}

// DispatchUntil fires the event until `acceptFunc` returns `true`, and returns the accepted result
// Returns the last result if no result is acceptable, and nil if no callback is called.
func (e *EventTarget) DispatchUntil(event *Event, acceptFunc func(any) bool) any {
	var ret any = nil
	for _, v := range e.val {
		ret = (*v)(event)
		if acceptFunc != nil && acceptFunc(ret) {
			break
		}
	}
	return ret
}
