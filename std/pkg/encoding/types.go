package encoding

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// Buffer is a buffer of bytes
type Buffer []byte

// Wire is a collection of Buffer. May be allocated in non-contiguous memory.
type Wire []Buffer

func (w Wire) Join() []byte {
	var lst = make([][]byte, len(w))
	for i, c := range w {
		lst[i] = c
	}
	ret := bytes.Join(lst, nil)
	if ret != nil {
		return ret
	} else {
		return []byte{}
	}
}

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

	Skip(n int) error

	Delegate(l int) ParseReader
}

type ErrUnrecognizedField struct {
	TypeNum TLNum
}

func (e ErrUnrecognizedField) Error() string {
	return fmt.Sprintf("There exists an unrecognized field that has a critical type number: %d", e.TypeNum)
}

var ErrBufferOverflow = errors.New("Buffer overflow when parsing. One of the TLV Length is wrong")

type ErrSkipRequired struct {
	TypeNum TLNum
}

func (e ErrSkipRequired) Error() string {
	return fmt.Sprintf("The required field of type %d is missing in the wire", e.TypeNum)
}

type ErrFailToParse struct {
	TypeNum TLNum
	Err     error
}

func (e ErrFailToParse) Error() string {
	return fmt.Sprintf("Failed to parse field %d: %v", e.TypeNum, e.Err)
}

func (e ErrFailToParse) Unwrap() error {
	return e.Err
}

// PlaceHolder is an empty structure that used to give names of procedure arguments.
type PlaceHolder struct{}
