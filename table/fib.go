/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

// FIBEntry Represents a FIB Entry
type FIBEntry struct {
	Prefix   string
	Nexthops []uint32
}

// FIB Holds the FIB
type FIB struct {
	Entries []FIBEntry
}
