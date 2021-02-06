/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

// TLV types for Management.
const (
	// Core
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

	// ForwarderStatus
	NfdVersion            = 0x80
	StartTimestamp        = 0x81
	CurrentTimestamp      = 0x82
	NNameTreeEntries      = 0x83
	NFibEntries           = 0x84
	NPitEntries           = 0x85
	NMeasurementEntries   = 0x86
	NCsEntries            = 0x87
	NInInterests          = 0x90
	NInData               = 0x91
	NInNacks              = 0x97
	NOutInterests         = 0x92
	NOutData              = 0x93
	NOutNacks             = 0x98
	NSatisfiedInterests   = 0x99
	NUnsatisfiedInterests = 0x9A

	// FaceMgmt
	FaceStatus            = 0x80
	ChannelStatus         = 0x82
	URIScheme             = 0x83
	FaceScope             = 0x84
	FacePersistency       = 0x85
	LinkType              = 0x86
	NInBytes              = 0x94
	NOutBytes             = 0x95
	FaceQueryFilter       = 0x96
	FaceEventNotification = 0xC0
	FaceEventKind         = 0xC1

	// FibMgmt
	FibEntry      = 0x80
	NextHopRecord = 0x81

	// CsMgmt
	CsInfo  = 0x80
	NHits   = 0x81
	NMisses = 0x82

	// StrategyMgmt
	StrategyChoice = 0x80

	// MeasurementStatus
	MeasurementEntry = 0x80
	StrategyInfo     = 0x81

	// RibMgmt
	RibEntry = 0x80
	Route    = 0x81
)
