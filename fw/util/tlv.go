/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package util

import "github.com/eric135/go-ndn/tlv"

// DecodeTypeLength decodes the TLV type and length from a byte slice.
func DecodeTypeLength(bytes []byte) (tlv.VarNum, tlv.VarNum, error) {
	var tlvType tlv.VarNum
	var tlvLength tlv.VarNum

	bytes, err := tlvType.Decode(bytes)
	if err != nil {
		return 0, 0, tlv.ErrIncomplete
	}

	_, err = tlvLength.Decode(bytes)
	if err != nil {
		return 0, 0, tlv.ErrIncomplete
	}

	return tlvType, tlvLength, nil
}
