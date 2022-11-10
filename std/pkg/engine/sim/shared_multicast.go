package sim

import (
	"errors"
	"math/rand"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type Link interface {
	PostPacket(enc.Wire, int) error
}

// DummyFace represents a transport that commutes with an engine
type DummyFace struct {
	link      Link
	linkToken int
	timer     ndn.Timer

	running bool
	onPkt   func(r enc.ParseReader) error
}

func (f *DummyFace) IsRunning() bool {
	return f.running
}

func (f *DummyFace) Open() error {
	if f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if f.running {
		return errors.New("face is already running")
	}
	f.running = true
	return nil
}

func (f *DummyFace) SetCallback(onPkt func(r enc.ParseReader) error, _ func(err error) error) {
	f.onPkt = onPkt
}

func (f *DummyFace) Close() error {
	if !f.running {
		return errors.New("face is not running")
	}
	f.running = false
	return nil
}

func (f *DummyFace) IsLocal() bool {
	return true // Mock local NFD
}

func (f *DummyFace) Send(pkt enc.Wire) error {
	// Handles all NFD packets
	// These are not forwarded to other nodes
	// For now we sends 200 OK to all requests
	parsed, _, err := spec.ReadPacket(enc.NewWireReader(pkt))
	if err == nil && parsed.Interest != nil {
		name := parsed.Interest.Name()
		prefix, _ := enc.NameFromStr("/localhost/nfd")
		if prefix.IsPrefix(name) {
			ctrlRes := mgmt.ControlResponse{
				Val: &mgmt.ControlArgs{
					StatusCode: utils.IdPtr[uint64](200),
					StatusText: utils.IdPtr("OK"),
				},
			}
			wire, _, _ := spec.Spec{}.MakeData(name, &ndn.DataConfig{}, ctrlRes.Encode(), sec.NewSha256Signer())
			if wire != nil {
				// Note: this must be scheduled, or it will block the channel in Engine.RegisterRoute()
				f.timer.Schedule(time.Duration(1), func() {
					f.onPkt(enc.NewWireReader(wire))
				})
			}
			return nil
		}
	}

	if !f.running {
		return errors.New("face is not running")
	}
	return f.link.PostPacket(pkt, f.linkToken)
}

func (f *DummyFace) FeedPacket(pkt enc.Wire) error {
	if !f.running {
		return errors.New("face is not running")
	}
	return f.onPkt(enc.NewWireReader(pkt))
}

// SharedMulticast represents a shared multicast media that forwards every data everywhere.
// Note: this is only designed to test the app, not for a real-world simulation.
// Therefore, things like queue size and bandwidth are considered as infinite.
// TODO: Add link down operation. Currently link down is simulated by loss rate.
// This needs to remember all scheduled events.
// TODO: Wrong design. Topology cannot be implemented. Current one is for PoC demo only.
type SharedMulticast struct {
	faces    []*DummyFace
	lossRate float32
	timer    ndn.Timer

	// delay is the propagation delay.
	delay time.Duration
}

func (l *SharedMulticast) PostPacket(pkt enc.Wire, token int) error {
	// Schedule event to handle propagation delay
	l.timer.Schedule(l.delay, func() {
		// Compute loss rate
		lossRand := rand.Float32()
		if lossRand < l.lossRate {
			// packet loss
			return
		}
		// Propagate the packet
		for i, f := range l.faces {
			if i == token {
				continue
			}
			f.FeedPacket(pkt)
		}
	})
	return nil
}

func (l *SharedMulticast) LossRate() float32 {
	return l.lossRate
}

func (l *SharedMulticast) SetLossRate(lossRate float32) {
	l.lossRate = lossRate
}

func (l *SharedMulticast) Delay() time.Duration {
	return l.delay
}

func (l *SharedMulticast) SetDelay(delay time.Duration) {
	l.delay = delay
}

func (l *SharedMulticast) Face(index int) basic_engine.Face {
	if index >= 0 && index < len(l.faces) {
		return l.faces[index]
	} else {
		return nil
	}
}

func NewSharedMulticast(timer ndn.Timer, numOfHosts int, lossRate float32, delay time.Duration) *SharedMulticast {
	ret := &SharedMulticast{
		faces:    make([]*DummyFace, numOfHosts),
		lossRate: lossRate,
		delay:    delay,
		timer:    timer,
	}
	for i := range ret.faces {
		ret.faces[i] = &DummyFace{
			link:      ret,
			linkToken: i,
			timer:     timer,
			running:   false,
		}
	}
	return ret
}
