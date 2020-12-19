// +build linux darwin dragonfly freebsd netbsd openbsd illumos solaris android aix
/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// SyscallReuseAddr sets SO_REUSEADDR on Unix-like platforms.
func SyscallReuseAddr(network string, address string, c syscall.RawConn) error {
	var err error
	c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	})
	return err
}
