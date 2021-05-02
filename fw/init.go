/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import "github.com/eric135/YaNFD/core"

// fwQueueSize is the maxmimum number of packets that can be buffered to be processed by a forwarding thread.
var fwQueueSize int

// NumFwThreads indicates the number of forwarding threads in the forwarder.
var NumFwThreads int

// lockThreadsToCores indicates whether forwarding threads will be locked to cores.
var lockThreadsToCores bool

// Configure configures the forwarding system.
func Configure() {
	fwQueueSize = core.GetConfigIntDefault("fw.queue_size", 1024)
	NumFwThreads = core.GetConfigIntDefault("fw.threads", 8)
	lockThreadsToCores = core.GetConfigBoolDefault("fw.lock_threads_to_cores", false)
}
