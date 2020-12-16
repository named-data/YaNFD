/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

import "errors"

// TLV errors.
var (
	ErrBufferTooShort       = errors.New("TLV length exceeds buffer size")
	ErrMissingLength        = errors.New("Missing TLV length")
	ErrUnexpected           = errors.New("Unexpected TLV type")
	ErrUnrecognizedCritical = errors.New("Unrecognized critical TLV type")
)
