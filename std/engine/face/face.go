package face

import enc "github.com/pulsejet/ndnd/std/encoding"

type Face interface {
	Open() error
	Close() error
	Send(pkt enc.Wire) error
	IsRunning() bool
	IsLocal() bool
	SetCallback(onPkt func(r enc.ParseReader) error,
		onError func(err error) error)
}
