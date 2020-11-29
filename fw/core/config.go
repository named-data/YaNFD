/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

// MaxNDNPacketSize is the maximum allowed NDN packet size
const MaxNDNPacketSize = 8800

// FaceQueueSize contains the maximum number of packets that can be buffered to be sent or received on a face
const FaceQueueSize = 128
