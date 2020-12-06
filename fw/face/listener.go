/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// Listener listens for incoming unicast connections for a transport type.
type Listener interface {
	String() string
	Run()
}

type listenerBase struct {
	HasQuit chan bool
}

func makeListenerBase() listenerBase {
	var l listenerBase
	l.HasQuit = make(chan bool)
	return l
}
