package basic

import (
	"bufio"
	"errors"
	"io"
	"net"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type StreamFace struct {
	network string
	addr    string
	local   bool
	conn    net.Conn
	running bool
	onPkt   func(r enc.ParseReader) error
	onError func(err error) error
}

func (f *StreamFace) Run() {
	r := bufio.NewReader(f.conn)
	for f.running {
		t, err := enc.ReadTLNum(r)
		if err != nil {
			if !f.running {
				break
			}
			err = f.onError(err)
			if err != nil {
				break
			}
		}
		l, err := enc.ReadTLNum(r)
		if err != nil {
			if !f.running {
				break
			}
			err = f.onError(err)
			if err != nil {
				break
			}
		}
		l0 := t.EncodingLength()
		l1 := l.EncodingLength()
		buf := make([]byte, l0+l1+int(l))
		t.EncodeInto(buf)
		l.EncodeInto(buf[l0:])
		_, err = io.ReadFull(r, buf[l0+l1:])
		if err != nil {
			if !f.running {
				break
			}
			err = f.onError(err)
			if err != nil {
				break
			}
		}
		err = f.onPkt(enc.NewBufferReader(buf))
		if err != nil {
			// Note: err returned by the engine's callback is used to interrupt the face loop
			// If it is recoverable, the engine should return log message and continue
			break
		}
	}
	f.running = false
	f.conn = nil
}

func (f *StreamFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if f.conn != nil {
		return errors.New("face is already running")
	}
	c, err := net.Dial(f.network, f.addr)
	if err != nil {
		return err
	}
	f.conn = c
	f.running = true
	go f.Run()
	return nil
}

func (f *StreamFace) Close() error {
	if f.conn == nil {
		return errors.New("face is not running")
	}
	f.running = false
	err := f.conn.Close()
	f.conn = nil
	return err
}

func (f *StreamFace) Send(pkt enc.Wire) error {
	if !f.running {
		return errors.New("face is not running")
	}
	for _, buf := range pkt {
		_, err := f.conn.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *StreamFace) IsRunning() bool {
	return f.running
}

func (f *StreamFace) IsLocal() bool {
	return f.local
}

func (f *StreamFace) SetCallback(onPkt func(r enc.ParseReader) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func NewStreamFace(network string, addr string, local bool) *StreamFace {
	return &StreamFace{
		network: network,
		addr:    addr,
		local:   local,
		onPkt:   nil,
		onError: nil,
		conn:    nil,
		running: false,
	}
}
