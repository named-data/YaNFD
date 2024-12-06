package ndn_defn

import (
	"encoding/binary"
	"errors"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

// Identical to go-ndn but returns error instead of panic
func ParseNat(buf enc.Buffer) (val enc.Nat, err error) {
	switch pos := len(buf); pos {
	case 1:
		return enc.Nat(buf[0]), nil
	case 2:
		return enc.Nat(binary.BigEndian.Uint16(buf)), nil
	case 4:
		return enc.Nat(binary.BigEndian.Uint32(buf)), nil
	case 8:
		return enc.Nat(binary.BigEndian.Uint64(buf)), nil
	default:
		return 0, errors.New("natural number length is not 1, 2, 4 or 8")
	}
}
