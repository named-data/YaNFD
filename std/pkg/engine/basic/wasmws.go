//go:build js && wasm

package basic

import (
	"errors"
	"net/url"
	"sync/atomic"
	"syscall/js"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
)

type WasmWsFace struct {
	network string
	addr    string
	local   bool
	conn    js.Value
	running atomic.Bool
	onPkt   func(r enc.ParseReader) error
	onError func(err error) error
}

func (f *WasmWsFace) onMessage(this js.Value, args []js.Value) any {
	event := args[0]
	data := event.Get("data")
	if !data.InstanceOf(js.Global().Get("ArrayBuffer")) {
		return nil
	}
	buf := make([]byte, data.Get("byteLength").Int())
	view := js.Global().Get("Uint8Array").New(data)
	js.CopyBytesToGo(buf, view)
	err := f.onPkt(enc.NewBufferReader(buf))
	if err != nil {
		f.running.Store(false)
		f.conn.Call("close")
		f.conn = js.Null()
	}
	return nil
}

func (f *WasmWsFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	l := pkt.Length()
	arr := js.Global().Get("Uint8Array").New(int(l))
	js.CopyBytesToJS(arr, pkt.Join())
	f.conn.Call("send", arr)
	return nil
}

func (f *WasmWsFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if !f.conn.IsNull() {
		return errors.New("face is already running")
	}
	u := url.URL{
		Scheme: f.network,
		Host:   f.addr,
		Path:   "/",
	}
	ch := make(chan struct{}, 1)
	// It seems now Go cannot handle exceptions thrown by JS
	f.conn = js.Global().Get("WebSocket").New(u.String())
	f.conn.Set("binaryType", "arraybuffer")
	f.conn.Call("addEventListener", "open", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ch <- struct{}{}
		close(ch)
		return nil
	}))
	f.conn.Call("addEventListener", "message", js.FuncOf(f.onMessage))
	log.WithField("module", "WasmWsFace").Info("Waiting for WebSocket connection ...")
	<-ch
	log.WithField("module", "WasmWsFace").Info("WebSocket connected ...")
	f.running.Store(true)
	return nil
}

func (f *WasmWsFace) Close() error {
	if f.conn.IsNull() {
		return errors.New("face is not running")
	}
	f.running.Store(false)
	f.conn.Call("close")
	f.conn = js.Null()
	return nil
}

func (f *WasmWsFace) IsRunning() bool {
	return f.running.Load()
}

func (f *WasmWsFace) IsLocal() bool {
	return f.local
}

func (f *WasmWsFace) SetCallback(onPkt func(r enc.ParseReader) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func NewWasmWsFace(network string, addr string, local bool) *WasmWsFace {
	return &WasmWsFace{
		network: network,
		addr:    addr,
		local:   local,
		onPkt:   nil,
		onError: nil,
		conn:    js.Null(),
		running: atomic.Bool{},
	}
}
