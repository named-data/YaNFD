package basic

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type StreamFace struct {
	network string
	addr    string
	local   bool
	conn    net.Conn
	running atomic.Bool
	onPkt   func(r enc.ParseReader) error
	onError func(err error) error
	sendMut sync.Mutex
}

func (f *StreamFace) Run() {
	r := bufio.NewReader(f.conn)
	for f.running.Load() {
		t, err := enc.ReadTLNum(r)
		if err != nil {
			if !f.running.Load() {
				break
			}
			err = f.onError(err)
			if err != nil {
				break
			}
		}
		l, err := enc.ReadTLNum(r)
		if err != nil {
			if !f.running.Load() {
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
			if !f.running.Load() {
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
	f.running.Store(false)
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
	f.running.Store(true)
	go f.Run()
	return nil
}

func (f *StreamFace) Close() error {
	if f.conn == nil {
		return errors.New("face is not running")
	}
	f.running.Store(false)
	err := f.conn.Close()
	// f.conn = nil // No need to do so, as Run() will set conn = nil
	return err
}

func (f *StreamFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	f.sendMut.Lock()
	defer f.sendMut.Unlock()
	for _, buf := range pkt {
		_, err := f.conn.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *StreamFace) IsRunning() bool {
	return f.running.Load()
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
		running: atomic.Bool{},
	}
}

type WebSocketFace struct {
	network string
	addr    string
	local   bool
	conn    *websocket.Conn
	running atomic.Bool
	onPkt   func(r enc.ParseReader) error
	onError func(err error) error
}

func (f *WebSocketFace) Run() {
	for f.running.Load() {
		messageType, pkt, err := f.conn.ReadMessage()
		if err != nil || messageType != websocket.BinaryMessage {
			// Ignore invalid message
			continue
		}
		err = f.onPkt(enc.NewBufferReader(pkt))
		if err != nil {
			// Note: err returned by the engine's callback is used to interrupt the face loop
			// If it is recoverable, the engine should return log message and continue
			break
		}
	}
	f.running.Store(false)
	f.conn = nil
}

func (f *WebSocketFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	return f.conn.WriteMessage(websocket.BinaryMessage, pkt.Join())
}

func (f *WebSocketFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if f.conn != nil {
		return errors.New("face is already running")
	}
	u := url.URL{
		Scheme: f.network,
		Host:   f.addr,
		Path:   "/",
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	f.conn = c
	f.running.Store(true)
	go f.Run()
	return nil
}

func (f *WebSocketFace) Close() error {
	if f.conn == nil {
		return errors.New("face is not running")
	}
	f.running.Store(false)
	err := f.conn.Close()
	// f.conn = nil // No need to do so, as Run() will set conn = nil
	return err
}

func (f *WebSocketFace) IsRunning() bool {
	return f.running.Load()
}

func (f *WebSocketFace) IsLocal() bool {
	return f.local
}

func (f *WebSocketFace) SetCallback(onPkt func(r enc.ParseReader) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func NewWebSocketFace(network string, addr string, local bool) *WebSocketFace {
	return &WebSocketFace{
		network: network,
		addr:    addr,
		local:   local,
		onPkt:   nil,
		onError: nil,
		conn:    nil,
		running: atomic.Bool{},
	}
}
