/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import "time"

// FaceQueueSize is the maximum number of packets that can be buffered to be sent or received on a face.
const FaceQueueSize = 1024

// FwQueueSize is the maxmimum number of packets that can be buffered to be processed by a forwarding thread.
const FwQueueSize = 1024

// DeadNonceListLifetime is the minimum lifetime of entries in the Dead Nonce List.
const DeadNonceListLifetime = 6 * time.Second // Value used by NFD.
