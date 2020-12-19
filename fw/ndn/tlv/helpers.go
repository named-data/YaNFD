/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/eric135/YaNFD/ndn/util"
)

// EncodeVarNum encodes a non-negative integer value for encoding.
func EncodeVarNum(in uint64) []byte {
	if in <= 0xFC {
		// This is just here to avoid having to write this condition in every other conditional.
		return []byte{byte(in)}
	} else if in <= 0xFFFF {
		bytes := make([]byte, 3)
		bytes[0] = 0xFD
		binary.BigEndian.PutUint16(bytes[1:], uint16(in))
		return bytes
	} else if in <= 0xFFFFFFFF {
		bytes := make([]byte, 5)
		bytes[0] = 0xFE
		binary.BigEndian.PutUint32(bytes[1:], uint32(in))
		return bytes
	} else {
		bytes := make([]byte, 9)
		bytes[0] = 0xFF
		binary.BigEndian.PutUint64(bytes[1:], in)
		return bytes
	}
}

// DecodeVarNum decodes a non-negative integer value from a wire value.
func DecodeVarNum(in []byte) (uint64, int, error) {
	if len(in) < 1 {
		return 0, 0, util.ErrTooShort
	}

	if in[0] <= 0xFC {
		return uint64(in[0]), 1, nil
	} else if in[0] == 0xFD {
		if len(in) < 3 {
			return 0, 0, util.ErrTooShort
		}
		return uint64(binary.BigEndian.Uint16(in[1:3])), 3, nil
	} else if in[0] == 0xFE {
		if len(in) < 5 {
			return 0, 0, util.ErrTooShort
		}
		return uint64(binary.BigEndian.Uint32(in[1:5])), 5, nil
	} else { // Must be 0xFF
		if len(in) < 9 {
			return 0, 0, util.ErrTooShort
		}
		return binary.BigEndian.Uint64(in[1:9]), 9, nil
	}
}

// EncodeNNIBlock encodes a non-negative integer value in a block of the specified type.
func EncodeNNIBlock(t uint32, v uint64) *Block {
	b := new(Block)
	b.SetType(t)
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, v)
	b.SetValue(value)
	return b
}

// DecodeNNIBlock decodes a non-negative integer value from a block.
func DecodeNNIBlock(wire *Block) (uint64, error) {
	if wire == nil {
		return 0, util.ErrNonExistent
	}
	if len(wire.Value()) != 8 {
		return 0, ErrBufferTooShort
	}
	return binary.BigEndian.Uint64(wire.Value()), nil
}

// DecodeTypeLength decodes the TLV type, TLV length, and total size of the block from a byte slice.
func DecodeTypeLength(bytes []byte) (uint32, int, int, error) {
	var tlvType uint64
	var tlvLength uint64

	tlvType, tlvTypeSize, err := DecodeVarNum(bytes)
	if err != nil {
		return 0, 0, 0, err
	} else if tlvType > math.MaxUint32 {
		return 0, 0, 0, errors.New("TLV type out of range")
	}

	tlvLength, tlvLengthSize, err := DecodeVarNum(bytes[tlvTypeSize:])
	if err != nil {
		return 0, 0, 0, err
	}

	return uint32(tlvType), int(tlvLength), tlvTypeSize + tlvLengthSize + int(tlvLength), nil
}
