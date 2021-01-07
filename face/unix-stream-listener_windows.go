/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// UnixStreamListener is a dummy listener for Windows.
type UnixStreamListener struct {
}

func (l *UnixStreamListener) Run() {
}

// MakeUnixStreamListener creates a dummy Unix stream listener for Windows.
func MakeUnixStreamListener(localURI URI) (*UnixStreamListener, error) {
	return new(UnixStreamListener), nil
}
