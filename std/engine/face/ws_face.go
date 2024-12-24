package face

import (
	"errors"
	"net/url"
	"sync/atomic"

	"github.com/gorilla/websocket"
	enc "github.com/named-data/ndnd/std/encoding"
)

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
