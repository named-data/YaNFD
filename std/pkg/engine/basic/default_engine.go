// Package basic gives a default implementation of the Engine interface.
// It only connects to local forwarding node via Unix socket.
package basic

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

const DefaultInterestLife = 4 * time.Second
const TimeoutMargin = 10 * time.Millisecond

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
	fib *NameTrie[fibEntry]

	// pit contains pending outgoing Interests.
	pit *NameTrie[pitEntry]

	// Since there is only one main coroutine, no need for RW locks.
	fibLock sync.Mutex
	pitLock sync.Mutex

	// log is used to log events, with "module=DefaultEngine". Need apex/log initialized.
	// Use WithField to set "name=".
	log *log.Entry

	// mgmtConf is the configuration for the management protocol.
	mgmtConf *mgmt.MgmtConfig

	// cmdChecker is used to validate NFD management packets.
	cmdChecker ndn.SigChecker
}

func (e *Engine) EngineTrait() ndn.Engine {
	return e
}

func (*Engine) Spec() ndn.Spec {
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
	var incomingFaceId *uint64 = nil
	var raw enc.Wire = nil

	if e.log.Level <= log.DebugLevel {
		wire := reader.Range(0, reader.Length())
		e.log.Debugf("Received packet bytes: %v", wire.Join())
	}

	pkt, ctx, err := spec.ReadPacket(reader)
	if err != nil {
		e.log.Errorf("Failed to parse packet: %v", err)
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
		incomingFaceId = lpPkt.IncomingFaceId
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
		e.onInterest(pkt.Interest, ndn.InterestHandlerExtra{
			RawInterest:    raw,
			SigCovered:     ctx.Interest_context.SigCovered(),
			PitToken:       pitToken,
			IncomingFaceId: incomingFaceId,
		})
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

func (e *Engine) onInterest(pkt *spec.Interest, extra ndn.InterestHandlerExtra) {
	// Compute deadline
	extra.Deadline = e.timer.Now()
	if pkt.InterestLifetimeV != nil {
		extra.Deadline = extra.Deadline.Add(*pkt.InterestLifetimeV)
	} else {
		extra.Deadline = extra.Deadline.Add(DefaultInterestLife)
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
		if extra.Deadline.Before(now) {
			e.log.WithField("name", pkt.NameV.String()).Warn("Deadline exceeded. Drop.")
			return ndn.ErrDeadlineExceed
		}
		if !e.face.IsRunning() {
			e.log.WithField("name", pkt.NameV.String()).Error("Cannot send through a closed face. Drop.")
			return ndn.ErrFaceDown
		}
		if extra.PitToken != nil {
			lpPkt := &spec.Packet{
				LpPacket: &spec.LpPacket{
					PitToken: extra.PitToken,
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

	// Call the handler. The handler should create goroutine to avoid blocking.
	// Do not `go` here because if Data is ready at hand, creating a go routine may be slower. Not tested though.
	handler(pkt, reply, extra)
}

func (e *Engine) onData(pkt *spec.Data, sigCovered enc.Wire, raw enc.Wire, pitToken []byte) {
	e.pitLock.Lock()
	defer e.pitLock.Unlock()
	n := e.pit.PrefixMatch(pkt.NameV)
	if n == nil {
		e.log.WithField("name", pkt.NameV.String()).Warn("Received Data for an unknown interest. Drop.")
		return
	}
	for cur := n; cur != nil; cur = cur.Parent() {
		curListSize := len(cur.Value())
		if curListSize <= 0 {
			continue
		}
		newList := make([]*pendInt, 0, curListSize)
		for _, entry := range cur.Value() {
			// CanBePrefix
			if cur.Depth() < len(pkt.NameV) && !entry.canBePrefix {
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
		return
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
	// TODO: Handle Interest cancellation
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

func (e *Engine) Express(
	finalName enc.Name, config *ndn.InterestConfig, rawInterest enc.Wire, callback ndn.ExpressCallbackFunc,
) error {
	var impSha256 []byte = nil
	var nodeName enc.Name = finalName

	if callback == nil {
		callback = func(ndn.InterestResult, ndn.Data, enc.Wire, enc.Wire, uint64) {}
	}

	// Handle implicit digest
	if len(finalName) <= 0 {
		return ndn.ErrInvalidValue{Item: "finalName", Value: finalName}
	}
	lastComp := finalName[len(finalName)-1]
	if lastComp.Typ == enc.TypeImplicitSha256DigestComponent {
		impSha256 = lastComp.Val
		nodeName = finalName[:len(finalName)-1]
	}

	// Handle deadline
	lifetime := DefaultInterestLife
	if config.Lifetime != nil {
		lifetime = *config.Lifetime
	}
	deadline := e.timer.Now().Add(lifetime)

	// Inject interest into PIT
	func() {
		e.pitLock.Lock()
		defer e.pitLock.Unlock()

		n := e.pit.MatchAlways(nodeName)
		timeoutFunc := func() {
			e.pitLock.Lock()
			defer e.pitLock.Unlock()
			now := e.timer.Now()
			lst := n.Value()
			newLst := make([]*pendInt, 0, len(lst))
			for _, entry := range lst {
				if entry.deadline.After(now) {
					newLst = append(newLst, entry)
				} else {
					if entry.callback != nil {
						entry.callback(ndn.InterestResultTimeout, nil, nil, nil, spec.NackReasonNone)
					} else {
						e.log.Fatalf("PIT has empty entry. This should not happen. Please check the implementation.")
					}
				}
			}
			n.SetValue(newLst)
			n.DeleteIf(func(lst []*pendInt) bool {
				return len(lst) == 0
			})
		}
		entry := &pendInt{
			callback:      callback,
			deadline:      deadline,
			canBePrefix:   config.CanBePrefix,
			mustBeFresh:   config.MustBeFresh,
			impSha256:     impSha256,
			timeoutCancel: e.timer.Schedule(lifetime+TimeoutMargin, timeoutFunc),
		}
		n.SetValue(append(n.Value(), entry))
	}()

	// Send interest
	err := e.face.Send(rawInterest)
	if err != nil {
		e.log.Errorf("Failed to send Interest: %v", err)
	} else if e.log.Level <= log.InfoLevel {
		e.log.WithField("name", finalName.String()).Info("Interest sent.")
	}

	return err
}

func (e *Engine) RegisterRoute(prefix enc.Name) error {
	intCfg := &ndn.InterestConfig{
		Lifetime: utils.IdPtr(1 * time.Second),
		Nonce:    utils.ConvertNonce(e.timer.Nonce()),
	}
	name, cmdWire, err := e.mgmtConf.MakeCmd("rib", "register", &mgmt.ControlArgs{Name: prefix}, intCfg)
	if err != nil {
		e.log.WithField("name", prefix.String()).Errorf("Failed to generate command Interest: %v", err)
		return err
	}
	ch := make(chan error)
	err = e.Express(name, intCfg, cmdWire,
		func(result ndn.InterestResult, data ndn.Data, rawData enc.Wire, sigCovered enc.Wire, nackReason uint64) {
			if result == ndn.InterestResultNack {
				ch <- fmt.Errorf("nack received: %v", nackReason)
			} else if result == ndn.InterestResultTimeout {
				ch <- ndn.ErrDeadlineExceed
			} else if result == ndn.InterestResultData {
				valid := e.cmdChecker(data.Name(), sigCovered, data.Signature())
				if !valid {
					ch <- fmt.Errorf("command signature is not valid")
				} else {
					ret, err := mgmt.ParseControlResponse(enc.NewWireReader(data.Content()), true)
					if err != nil {
						ch <- err
					} else {
						if ret.Val != nil {
							if ret.Val.StatusCode == 200 {
								ch <- nil
							} else {
								errText := ret.Val.StatusText
								ch <- fmt.Errorf("registration failed due to error %d: %s", ret.Val.StatusCode, errText)
							}
						} else {
							ch <- fmt.Errorf("improper response")
						}
					}
				}
			} else {
				ch <- fmt.Errorf("unknown result: %v", result)
			}
			close(ch)
		})
	if err != nil {
		e.log.WithField("name", prefix.String()).Errorf("Failed to express command Interest: %v", err)
		return err
	}
	err = <-ch
	if err != nil {
		e.log.WithField("name", prefix.String()).Errorf("Failed to register prefix: %v", err)
		return err
	} else {
		e.log.WithField("name", prefix.String()).Info("Prefix registered.")
	}
	return nil
}

func (e *Engine) UnregisterRoute(prefix enc.Name) error {
	intCfg := &ndn.InterestConfig{
		Lifetime: utils.IdPtr(1 * time.Second),
		Nonce:    utils.ConvertNonce(e.timer.Nonce()),
	}
	name, cmdWire, err := e.mgmtConf.MakeCmd("rib", "register", &mgmt.ControlArgs{Name: prefix}, intCfg)
	if err != nil {
		e.log.WithField("name", prefix.String()).Errorf("Failed to generate command Interest: %v", err)
		return err
	}
	err = e.Express(name, intCfg, cmdWire, nil)
	if err != nil {
		e.log.WithField("name", prefix.String()).Errorf("Failed to express command Interest: %v", err)
		return err
	} else {
		e.log.WithField("name", prefix.String()).Info("Prefix unregistered.")
	}
	return nil
}

func NewEngine(face Face, timer ndn.Timer, cmdSigner ndn.Signer, cmdChecker ndn.SigChecker) *Engine {
	if face == nil || timer == nil || cmdSigner == nil || cmdChecker == nil {
		return nil
	}
	logger := log.WithField("module", "basic_engine")
	mgmtCfg := mgmt.NewConfig(face.IsLocal(), cmdSigner, spec.Spec{})
	return &Engine{
		face:       face,
		timer:      timer,
		mgmtConf:   mgmtCfg,
		cmdChecker: cmdChecker,
		log:        logger,
		fib:        NewNameTrie[fibEntry](),
		pit:        NewNameTrie[pitEntry](),
		fibLock:    sync.Mutex{},
		pitLock:    sync.Mutex{},
	}
}
