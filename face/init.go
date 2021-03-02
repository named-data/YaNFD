/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import "github.com/eric135/YaNFD/core"

// faceQueueSize is the maximum number of packets that can be buffered to be sent or received on a face.
var faceQueueSize int

// Configure configures the face system.
func Configure() {
	faceQueueSize = core.GetConfigIntDefault("faces.queue_size", 1024)
}
