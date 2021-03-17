/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package util

import "errors"

// NDN common errors.
var (
	ErrDecodeNameComponent = errors.New("error decoding name component")
	ErrNonExistent         = errors.New("required value does not exist")
	ErrOutOfRange          = errors.New("value outside of allowed range")
	ErrTooLong             = errors.New("value too long")
	ErrTooShort            = errors.New("value too short")
)
