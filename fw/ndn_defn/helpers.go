package ndn_defn

import (
	"bytes"
	"io"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

func DecodeTypeLength(buf []byte) (enc.TLNum, enc.TLNum, error) {
	return ReadTypeLength(bytes.NewReader(buf))
}

func ReadTypeLength(reader io.ByteReader) (enc.TLNum, enc.TLNum, error) {
	typ, err := enc.ReadTLNum(reader)
	if err != nil {
		return 0, 0, err
	}

	len, err := enc.ReadTLNum(reader)
	if err != nil {
		return 0, 0, err
	}

	return typ, len, nil
}
