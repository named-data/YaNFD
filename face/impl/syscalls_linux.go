// +build linux

/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"strconv"
	"syscall"

	"github.com/eric135/YaNFD/core"
	"golang.org/x/sys/unix"
)

// SyscallGetSocketSendQueueSize returns the current size of the send queue on the specified socket.
func SyscallGetSocketSendQueueSize(c syscall.RawConn) uint64 {
	var val int
	c.Control(func(fd uintptr) {
		var err error
		val, err = unix.IoctlGetInt(int(fd), unix.SIOCOUTQ)
		if err != nil {
			core.LogWarn("Face-Syscall", "Unable to get size of socket send queue for fd="+strconv.Itoa(int(fd))+": "+err.Error())
			val = 0
		}
	})
	return uint64(val)
}
