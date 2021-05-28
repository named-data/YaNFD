// +build !windows,!cgo

/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package impl

import (
	"errors"

	"github.com/eric135/YaNFD/core"
)

// OpenPcap returns an error on unsupported platform.
func OpenPcap(device, bpfFilter string) (PcapHandle, error) {
	core.LogError("Face-Pcap", "PCAP not supported on this platform")
	return nil, errors.New("pcap not supported on this platform")
}
