/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv

// TLV types for NDN.
const (
	// Packet types
	Interest = 0x05
	Data     = 0x06

	// Name and components
	Name                            = 0x07
	ImplicitSha256DigestComponent   = 0x01
	ParametersSha256DigestComponent = 0x02
	GenericNameComponent            = 0x08
	KeywordNameComponent            = 0x20
	SegmentNameComponent            = 0x21
	ByteOffsetNameComponent         = 0x22
	VersionNameComponent            = 0x23
	TimestampNameComponent          = 0x24
	SequenceNumNameComponent        = 0x25

	// Interest packets
	CanBePrefix            = 0x21
	MustBeFresh            = 0x12
	ForwardingHint         = 0x1e
	Nonce                  = 0x0a
	InterestLifetime       = 0x0c
	HopLimit               = 0x22
	ApplicationParameters  = 0x24
	InterestSignatureInfo  = 0x2c
	InterestSignatureValue = 0x2e

	// Data packets
	MetaInfo       = 0x14
	Content        = 0x15
	SignatureInfo  = 0x16
	SignatureValue = 0x17

	// Data/MetaInfo
	ContentType     = 0x18
	FreshnessPeriod = 0x19
	FinalBlockID    = 0x1a

	// Signature
	SignatureType   = 0x1b
	KeyLocator      = 0x1c
	KeyDigest       = 0x1d
	SignatureNonce  = 0x26
	SignatureTime   = 0x28
	SignatureSeqNum = 0x2a

	// Link Object
	Delegation = 0x1f
	Preference = 0x1e
)

// IsCritical returns whether a TLV type is critical.
func IsCritical(tlvType uint32) bool {
	if tlvType < 0x20 {
		return true
	}
	return tlvType&0x1 == 1
}
