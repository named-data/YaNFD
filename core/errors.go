/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import "errors"

// Error definitions
var (
	ErrNotCanonical = errors.New("URI could not be canonized")
)
