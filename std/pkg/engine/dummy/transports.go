package dummy

import (
	"errors"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type DummyFace struct {
	sendPkts []enc.Buffer
	running  bool
	onPkt    func(r enc.ParseReader) error
	onError  func(err error) error
}

func (f *DummyFace) IsRunning() bool {
	return f.running
}

func (f *DummyFace) IsLocal() bool {
	return true
}

func (f *DummyFace) SetCallback(onPkt func(r enc.ParseReader) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func (f *DummyFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("Face callbacks are not set")
	}
	if !f.running {
		return errors.New("Face is already running")
	}
	f.sendPkts = make([]enc.Buffer, 0)
	f.running = true
	return nil
}

func (f *DummyFace) Close() error {
	if !f.running {
		return errors.New("Face is not running")
	}
	f.running = false
	return nil
}

func (f *DummyFace) RecvPacket(pkt enc.Buffer) error {
	if !f.running {
		return errors.New("Face is not running")
	}
	return f.onPkt(enc.NewBufferReader(pkt))
}

func (f *DummyFace) Consume() (enc.Buffer, error) {
	if !f.running {
		return nil, errors.New("Face is not running")
	}
	if len(f.sendPkts) == 0 {
		return nil, errors.New("No packet to consume")
	}
	pkt := f.sendPkts[0]
	f.sendPkts = f.sendPkts[1:]
	return pkt, nil
}

func (f *DummyFace) Send(pkt enc.Wire) error {
	if !f.running {
		return errors.New("Face is not running")
	}
	if len(pkt) == 1 {
		f.sendPkts = append(f.sendPkts, pkt[0])
	} else if len(pkt) >= 2 {
		newBuf := make(enc.Buffer, 0)
		for _, buf := range pkt {
			newBuf = append(newBuf, buf...)
		}
		f.sendPkts = append(f.sendPkts, newBuf)
	}
	return nil
}

func NewDummyFace() *DummyFace {
	return &DummyFace{}
}
