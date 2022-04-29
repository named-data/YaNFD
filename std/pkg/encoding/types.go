package encoding

import (
	"io"
)

// Buffer is a buffer of bytes
type Buffer []byte

// Wire is a collection of Buffer. May be allocated in non-contiguous memory.
type Wire []Buffer

type ErrFormat struct {
	Msg string
}

func (e ErrFormat) Error() string {
	return e.Msg
}

type ErrNotFound struct {
	Key string
}

func (e ErrNotFound) Error() string {
	return e.Key + ": not found"
}

// ParseReader is an interface operating on Buffer and Wire
type ParseReader interface {
	io.Reader
	io.ByteScanner

	// ReadWire reads a list of buffers in place without copy.
	// It always tries to read the required length of bytes.
	ReadWire(l int) (Wire, error)

	Range(start, end int) Wire

	Pos() int

	Length() int
}
