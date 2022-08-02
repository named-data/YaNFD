// Package basic gives a default implementation of the Engine interface.
// It only connects to local forwarding node via Unix socket.
package basic

import (
	"sync"
	"time"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
)

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
	callback      ndn.ExpressCallbackFunc
	deadline      time.Time
	canBePrefix   bool
	mustBeFresh   bool
	impSha256     []byte
	timeoutCancel func() error
}

type pitEntry = []*pendInt

type Engine struct {
	face  Face
	timer ndn.Timer
	fib   NameTrie[fibEntry]
	pit   NameTrie[pitEntry]

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
	pkt, pc, err := spec.ReadPacket(reader)
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
	// TODO
}

func (e *Engine) onError(err error) error {
	// TODO
}

func (e *Engine) mainLoop() error {
	e.log.Info("Default engine started.")
	e.face.SetCallback(e.onPacket, e.onError)
	err := e.face.Open()
	if err != nil {
		e.log.Errorf("Face failed to open: %v", err)
		return err
	}
	// TODO
}

func (e *Engine) Express(finalName enc.Name, config *ndn.InterestConfig,
	rawInterest enc.Wire, callback ndn.ExpressCallbackFunc) error {

}

func (e *Engine) RegisterRoute(prefix enc.Name) error {
}

func (e *Engine) UnregisterRoute(prefix enc.Name) error {

}

// Command validator is delayed to future work. For localhost, check local key. For localhop, user provide public key.
