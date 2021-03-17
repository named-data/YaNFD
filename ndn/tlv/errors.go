/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

import "errors"

// TLV errors.
var (
	ErrBufferTooShort       = errors.New("TLV length exceeds buffer size")
	ErrMissingLength        = errors.New("missing TLV length")
	ErrUnexpected           = errors.New("unexpected TLV type")
	ErrUnrecognizedCritical = errors.New("unrecognized critical TLV type")
)
