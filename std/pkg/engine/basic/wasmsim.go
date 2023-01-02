//go:build js && wasm

package basic

import (
	"errors"
	"sync/atomic"
	"syscall/js"

	"github.com/apex/log"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type WasmSimFace struct {
	gosim   js.Value
	running atomic.Bool
	onPkt   func(r enc.ParseReader) error
	onError func(err error) error
}

func (f *WasmSimFace) onMessage(this js.Value, args []js.Value) any {
	pkt := args[0]
	if !pkt.InstanceOf(js.Global().Get("Uint8Array")) {
		return nil
	}
	buf := make([]byte, pkt.Get("byteLength").Int())
	js.CopyBytesToGo(buf, pkt)
	err := f.onPkt(enc.NewBufferReader(buf))
	if err != nil {
		f.running.Store(false)
		log.Errorf("Unable to handle packet: %+v", err)
	}
	return nil
}

func (f *WasmSimFace) Send(pkt enc.Wire) error {
	if !f.running.Load() {
		return errors.New("face is not running")
	}
	l := pkt.Length()
	arr := js.Global().Get("Uint8Array").New(int(l))
	js.CopyBytesToJS(arr, pkt.Join())
	f.gosim.Call("sendPkt", arr)
	return nil
}

func (f *WasmSimFace) Open() error {
	if f.onError == nil || f.onPkt == nil {
		return errors.New("face callbacks are not set")
	}
	if !f.gosim.IsNull() {
		return errors.New("face is already running")
	}
	// It seems now Go cannot handle exceptions thrown by JS
	f.gosim = js.Global().Get("gondnsim")
	f.gosim.Call("setRecvPktCallback", js.FuncOf(f.onMessage))
	log.WithField("module", "WasmSimFace").Info("Sim face started.")
	f.running.Store(true)
	return nil
}

func (f *WasmSimFace) Close() error {
	if f.gosim.IsNull() {
		return errors.New("face is not running")
	}
	f.running.Store(false)
	f.gosim.Call("setRecvPktCallback", js.FuncOf(func(this js.Value, args []js.Value) any {
		return nil
	}))
	f.gosim = js.Null()
	return nil
}

func (f *WasmSimFace) IsRunning() bool {
	return f.running.Load()
}

func (f *WasmSimFace) IsLocal() bool {
	return true
}

func (f *WasmSimFace) SetCallback(onPkt func(r enc.ParseReader) error,
	onError func(err error) error) {
	f.onPkt = onPkt
	f.onError = onError
}

func NewWasmSimFace() *WasmSimFace {
	return &WasmSimFace{
		onPkt:   nil,
		onError: nil,
		gosim:   js.Null(),
		running: atomic.Bool{},
	}
}
