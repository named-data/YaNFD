/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package util

import "errors"

// NDN common errors.
var (
	ErrDecodeNameComponent = errors.New("Error decoding name component")
	ErrNonExistent         = errors.New("Required value does not exist")
	ErrOutOfRange          = errors.New("Value outside of allowed range")
	ErrTooLong             = errors.New("Value too long")
	ErrTooShort            = errors.New("Value too short")
)
