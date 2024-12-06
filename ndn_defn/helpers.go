package ndn_defn

import (
	"bytes"
	"io"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

func DecodeTypeLength(buf []byte) (uint32, int, int, error) {
	return ReadTypeLength(bytes.NewReader(buf))
}

func ReadTypeLength(reader io.ByteReader) (uint32, int, int, error) {
	typ, err := enc.ReadTLNum(reader)
	if err != nil {
		return 0, 0, 0, err
	}

	len, err := enc.ReadTLNum(reader)
	if err != nil {
		return 0, 0, 0, err
	}

	tlvSize := typ.EncodingLength() + len.EncodingLength() + int(len)

	return uint32(typ), int(len), tlvSize, nil
}
