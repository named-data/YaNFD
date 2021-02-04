/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

// TLV types for NDN.
const (
	ControlParameters             = 0x68
	FaceID                        = 0x69
	URI                           = 0x72
	LocalURI                      = 0x81
	Origin                        = 0x6F
	Cost                          = 0x6A
	Capacity                      = 0x83
	Count                         = 0x84
	BaseCongestionMarkingInterval = 0x87
	DefaultCongestionThreshold    = 0x88
	MTU                           = 0x89
	Flags                         = 0x6C
	Mask                          = 0x70
	Strategy                      = 0x6B
	ExpirationPeriod              = 0x6D
	ControlResponse               = 0x65
	StatusCode                    = 0x66
	StatusText                    = 0x67
)
