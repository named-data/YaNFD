package misc

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// This benchmark is used to show that ByteReader is faster than Reader.

func ReadTLNumReader(r io.Reader) (val uint64, err error) {
	buf := make([]byte, 1, 9)
	if _, err = io.ReadFull(r, buf); err != nil {
		return
	}
	switch x := buf[0]; {
	case x <= 0xfc:
		val = uint64(x)
	case x == 0xfd:
		if _, err = io.ReadFull(r, buf[1:3]); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return
		}
		val = uint64(binary.BigEndian.Uint16(buf[1:3]))
	case x == 0xfe:
		if _, err = io.ReadFull(r, buf[1:5]); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return
		}
		val = uint64(binary.BigEndian.Uint32(buf[1:5]))
	case x == 0xff:
		if _, err = io.ReadFull(r, buf[1:9]); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return
		}
		val = uint64(binary.BigEndian.Uint64(buf[1:9]))
	}
	return
}

func ReadTLNumByteReader(r io.ByteReader) (val uint64, err error) {
	var x byte
	if x, err = r.ReadByte(); err != nil {
		return
	}
	l := 1
	switch {
	case x <= 0xfc:
		val = uint64(x)
		return
	case x == 0xfd:
		l = 2
	case x == 0xfe:
		l = 4
	case x == 0xff:
		l = 8
	}
	val = 0
	for i := 0; i < l; i++ {
		if x, err = r.ReadByte(); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return
		}
		val = val<<8 | uint64(x)
	}
	return
}

func BenchmarkTestRead(b *testing.B) {
	buf := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0xfd, 0x01, 0x02, 0xfd, 0x04, 0x05, 0xfd, 0x07, 0x08, 0xfd, 0x0a, 0x0b, 0xfd, 0x0d, 0x0e, 0xfd, 0x0f, 0x0f,
		0xfe, 0x01, 0x02, 0x03, 0x04, 0xfe, 0x06, 0x07, 0x08, 0x09, 0xfe, 0x0b, 0x0c, 0x0d, 0x0e,
		0xfe, 0x0f, 0x0f, 0x0f, 0x0f,
		0xff, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0xff, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	}
	b.Run(
		b.Name()+": Buffer",
		func(b *testing.B) {
			r := bytes.NewBuffer(buf)
			for i := 0; i < 28; i++ {
				_, _ = ReadTLNumReader(r)
			}
		},
	)
	b.Run(
		b.Name()+": Buffer (Byte)",
		func(b *testing.B) {
			r := bytes.NewBuffer(buf)
			for i := 0; i < 28; i++ {
				_, _ = ReadTLNumByteReader(r)
			}
		},
	)
	b.Run(
		b.Name()+": enc.Buffer",
		func(b *testing.B) {
			r := enc.NewBufferReader(buf)
			for i := 0; i < 28; i++ {
				_, _ = ReadTLNumReader(r)
			}
		},
	)
	b.Run(
		b.Name()+": enc.Buffer (Byte)",
		func(b *testing.B) {
			r := enc.NewBufferReader(buf)
			for i := 0; i < 28; i++ {
				_, _ = ReadTLNumByteReader(r)
			}
		},
	)
	b.Run(
		b.Name()+": enc.Wire",
		func(b *testing.B) {
			r := enc.NewWireReader(enc.Wire{buf})
			for i := 0; i < 28; i++ {
				_, _ = ReadTLNumReader(r)
			}
		},
	)
	b.Run(
		b.Name()+": enc.Wire (Byte)",
		func(b *testing.B) {
			r := enc.NewWireReader(enc.Wire{buf})
			for i := 0; i < 28; i++ {
				_, _ = ReadTLNumByteReader(r)
			}
		},
	)
}
