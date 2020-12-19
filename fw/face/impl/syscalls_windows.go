// +build windows
/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// SyscallReuseAddr sets SO_REUSEADDR on Windows.
func SyscallReuseAddr(network string, address string, c syscall.RawConn) error {
	var err error
	c.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	})
	return err
}
