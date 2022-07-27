// Package basic gives a default implementation of the Engine interface.
// It only connects to local forwarding node via Unix socket.
package basic

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
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

type Engine struct {
	face  Face
	timer ndn.Timer
	// fib
	// pit
}

func (e *Engine) EngineTrait() ndn.Engine {
	return e
}

func (_ *Engine) Spec() ndn.Spec {
	return spec_2022.Spec{}
}

func (e *Engine) Timer() ndn.Timer {
	return e.timer
}

func (e *Engine) AttachHandler(prefix enc.Name, handler ndn.InterestHandler) error {
}

func (e *Engine) DetachHandler(prefix enc.Name) error {

}

func (e *Engine) Express(finalName enc.Name, config *ndn.InterestConfig,
	rawInterest enc.Wire, callback ndn.ExpressCallbackFunc) error {

}

func (e *Engine) RegisterRoute(prefix enc.Name) error {
}

func (e *Engine) UnregisterRoute(prefix enc.Name) error {

}
