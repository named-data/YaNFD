// Package basic gives a default implementation of the Engine interface.
// It only connects to local forwarding node via Unix socket.
package basic

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
)

const DefaultInterestLife = 4 * time.Second

type Face interface {
	Open() error
	Close() error
	Send(pkt enc.Wire) error
	IsRunning() bool
	IsLocal() bool
	SetCallback(onPkt func(r enc.ParseReader) error,
		onError func(err error) error)
}

type fibEntry = ndn.InterestHandler

type pendInt struct {
	callback    ndn.ExpressCallbackFunc
	deadline    time.Time
	canBePrefix bool
	// mustBeFresh is actually not useful, since Freshness is decided by the cache, not us.
	mustBeFresh   bool
	impSha256     []byte
	timeoutCancel func() error
}

type pitEntry = []*pendInt

type Engine struct {
	face  Face
	timer ndn.Timer

	// fib contains the registered Interest handlers.
	fib NameTrie[fibEntry]

	// pit contains pending outgoing Interests.
	pit NameTrie[pitEntry]

	// Since there is only one main coroutine, no need for RW locks.
	fibLock sync.Mutex
	pitLock sync.Mutex

	// log is used to log events, with "module=DefaultEngine". Need apex/log initialized.
	// Use WithField to set "name=".
	log log.Entry
}

func (e *Engine) EngineTrait() ndn.Engine {
	return e
}

func (_ *Engine) Spec() ndn.Spec {
	return spec.Spec{}
}

func (e *Engine) Timer() ndn.Timer {
	return e.timer
}

func (e *Engine) AttachHandler(prefix enc.Name, handler ndn.InterestHandler) error {
	e.fibLock.Lock()
	defer e.fibLock.Unlock()

	pred := func(cb fibEntry) bool {
		return cb != nil
	}
	n := e.fib.FirstSatisfyOrNew(prefix, pred)
	if n.Value() != nil || n.HasChildren() {
		return ndn.ErrPrefixPropViolation
	}
	n.SetValue(handler)
	return nil
}

func (e *Engine) DetachHandler(prefix enc.Name) error {
	e.fibLock.Lock()
	defer e.fibLock.Unlock()

	n := e.fib.ExactMatch(prefix)
	if n == nil {
		return ndn.ErrInvalidValue{Item: "prefix", Value: prefix}
	}
	n.Delete()
	return nil
}

func (e *Engine) onPacket(reader enc.ParseReader) error {
	var nackReason uint64 = spec.NackReasonNone
	var pitToken []byte = nil
	var raw enc.Wire = nil

	pkt, ctx, err := spec.ReadPacket(reader)
	if err != nil {
		e.log.Errorf("Failed to parse packet: %v", err)
		if e.log.Level <= log.DebugLevel {
			wire := reader.Range(0, reader.Length())
			e.log.Debugf("Failed packet bytes: %v", wire.Join())
		}
		// Recoverable error. Should continue.
		return nil
	}
	// Now, exactly one of Interest, Data, LpPacket is not nil
	// First check LpPacket, and do further parse.
	if pkt.LpPacket != nil {
		lpPkt := pkt.LpPacket
		if lpPkt.FragIndex != nil || lpPkt.FragCount != nil {
			e.log.Warnf("Fragmented LpPackets are not supported. Drop.")
			return nil
		}
		// Parse the inner packet.
		raw = pkt.LpPacket.Fragment
		if len(raw) == 1 {
			pkt, ctx, err = spec.ReadPacket(enc.NewBufferReader(raw[0]))
		} else {
			pkt, ctx, err = spec.ReadPacket(enc.NewWireReader(raw))
		}
		if err != nil || (pkt.Data == nil) == (pkt.Interest == nil) {
			e.log.Errorf("Failed to parse packet in LpPacket: %v", err)
			if e.log.Level <= log.DebugLevel {
				wire := reader.Range(0, reader.Length())
				e.log.Debugf("Failed packet bytes: %v", wire.Join())
			}
			// Recoverable error. Should continue.
			return nil
		}
		// Set parameters
		if lpPkt.Nack != nil {
			nackReason = lpPkt.Nack.Reason
		}
		pitToken = lpPkt.PitToken
	} else {
		raw = reader.Range(0, reader.Length())
	}
	// Now pkt is either Data or Interest (including Nack).
	if nackReason != spec.NackReasonNone {
		if pkt.Interest == nil {
			e.log.Errorf("Received nack for an Data")
			return nil
		}
		if e.log.Level <= log.InfoLevel {
			nameStr := pkt.Interest.NameV.String()
			e.log.WithField("name", nameStr).Infof("Nack received for %v", nackReason)
		}
		e.onNack(pkt.Interest.NameV, nackReason)
	} else if pkt.Interest != nil {
		if e.log.Level <= log.InfoLevel {
			nameStr := pkt.Interest.NameV.String()
			e.log.WithField("name", nameStr).Info("Interest received.")
		}
		e.onInterest(pkt.Interest, ctx.Interest_context.SigCovered(), raw, pitToken)
	} else if pkt.Data != nil {
		if e.log.Level <= log.InfoLevel {
			nameStr := pkt.Data.NameV.String()
			e.log.WithField("name", nameStr).Info("Data received.")
		}
		// PitToken is not used for now
		e.onData(pkt.Data, ctx.Data_context.SigCovered(), raw, pitToken)
	} else {
		log.Fatalf("Unreachable. Check spec implementation.")
	}
	return nil
}

func (e *Engine) onInterest(pkt *spec.Interest, sigCovered enc.Wire, raw enc.Wire, pitToken []byte) {
	// Compute deadline
	deadline := e.timer.Now()
	if pkt.InterestLifetimeV != nil {
		deadline = deadline.Add(*pkt.InterestLifetimeV)
	} else {
		deadline = deadline.Add(DefaultInterestLife)
	}

	// Match node
	handler := func() ndn.InterestHandler {
		e.fibLock.Lock()
		defer e.fibLock.Unlock()
		n := e.fib.PrefixMatch(pkt.NameV)
		// We can directly return because of the prefix-free condition
		return n.Value()
		// If it does not hold, us the following:
		// for n != nil && n.Value() == nil {
		// 	n = n.Parent()
		// }
		// if n == nil {
		// 	return nil
		// } else {
		// 	return n.Value()
		// }
	}()
	if handler == nil {
		e.log.WithField("name", pkt.NameV.String()).Warn("No handler. Drop.")
		return
	}

	// The reply callback function
	reply := func(encodedData enc.Wire) error {
		now := e.timer.Now()
		if deadline.Before(now) {
			e.log.WithField("name", pkt.NameV.String()).Warn("Deadline exceeded. Drop.")
			return ndn.ErrDeadlineExceed
		}
		if !e.face.IsRunning() {
			e.log.WithField("name", pkt.NameV.String()).Error("Cannot send through a closed face. Drop.")
			return ndn.ErrFaceDown
		}
		if pitToken != nil {
			lpPkt := &spec.Packet{
				LpPacket: &spec.LpPacket{
					PitToken: pitToken,
					Fragment: encodedData,
				},
			}
			encoder := spec.PacketEncoder{}
			encoder.Init(lpPkt)
			wire := encoder.Encode(lpPkt)
			if wire == nil {
				return ndn.ErrFailedToEncode
			}
			return e.face.Send(wire)
		} else {
			return e.face.Send(encodedData)
		}
	}

	// Call the handler. Create goroutine to avoid blocking.
	go handler(pkt, raw, sigCovered, reply, deadline)
}

func (e *Engine) onData(pkt *spec.Data, sigCovered enc.Wire, raw enc.Wire, pitToken []byte) {
	e.pitLock.Lock()
	defer e.pitLock.Unlock()
	n := e.pit.ExactMatch(pkt.NameV)
	if n == nil {
		e.log.WithField("name", pkt.NameV.String()).Warn("Received Data for an unknown interest. Drop.")
	}
	for cur := n; cur != nil; cur = cur.Parent() {
		curListSize := len(cur.Value())
		if curListSize <= 0 {
			continue
		}
		newList := make([]*pendInt, 0, curListSize)
		for _, entry := range cur.Value() {
			// CanBePrefix
			if n.Depth() < len(pkt.NameV) && !entry.canBePrefix {
				newList = append(newList, entry)
				continue
			}
			// We don't check MustBeFresh, as it is the job of the cache/forwarder.
			// ImplicitDigest256
			if entry.impSha256 != nil {
				h := sha256.New()
				for _, buf := range raw {
					h.Write(buf)
				}
				digest := h.Sum(nil)
				if !bytes.Equal(entry.impSha256, digest) {
					newList = append(newList, entry)
					continue
				}
			}
			// entry satisfied
			entry.timeoutCancel()
			if entry.callback == nil {
				e.log.Fatalf("PIT has empty entry. This should not happen. Please check the implementation.")
				continue
			}
			entry.callback(ndn.InterestResultData, pkt, raw, sigCovered, spec.NackReasonNone)
		}
		cur.SetValue(newList)
	}
	n.DeleteIf(func(lst []*pendInt) bool {
		return len(lst) == 0
	})
}

func (e *Engine) onNack(name enc.Name, reason uint64) {
	e.pitLock.Lock()
	defer e.pitLock.Unlock()
	n := e.pit.ExactMatch(name)
	if n == nil {
		e.log.WithField("name", name.String()).Warn("Received Nack for an unknown interest. Drop.")
	}
	for _, entry := range n.Value() {
		entry.timeoutCancel()
		if entry.callback != nil {
			entry.callback(ndn.InterestResultNack, nil, nil, nil, reason)
		} else {
			e.log.Fatalf("PIT has empty entry. This should not happen. Please check the implementation.")
		}
	}
	n.Delete()
}

func (e *Engine) onError(err error) error {
	e.log.Errorf("Error on face, quit: %v", err)
	return err
}

func (e *Engine) Start() error {
	if e.face.IsRunning() {
		return errors.New("Face is already running")
	}
	e.log.Info("Default engine start.")
	e.face.SetCallback(e.onPacket, e.onError)
	err := e.face.Open()
	if err != nil {
		e.log.Errorf("Face failed to open: %v", err)
		return err
	}
	return nil
}

func (e *Engine) Shutdown() error {
	if !e.face.IsRunning() {
		return errors.New("Face is not running")
	}
	e.log.Info("Default engine shutdown.")
	return e.face.Close()
}

func (e *Engine) Express(finalName enc.Name, config *ndn.InterestConfig,
	rawInterest enc.Wire, callback ndn.ExpressCallbackFunc) error {
	return nil // TODO
}

func (e *Engine) RegisterRoute(prefix enc.Name) error {
	return nil // TODO
}

func (e *Engine) UnregisterRoute(prefix enc.Name) error {
	return nil // TODO
}

// Command validator is delayed to future work. For localhost, check local key. For localhop, user provide public key.
