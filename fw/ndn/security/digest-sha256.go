/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package security

import (
	"crypto/sha256"
	"crypto/subtle"
)

// DigestSha256 represents a signer that performs a SHA-256 digest over the packet.
type DigestSha256 struct {
}

var _ Signer = &DigestSha256{}

// Sign signs a buffer using DigestSha256.
func (d *DigestSha256) Sign(buf []byte) ([]byte, error) {
	sum := sha256.Sum256(buf)
	return sum[:], nil
}

// Validate returns whether the provided signature is valid for the provided buffer.
func (d *DigestSha256) Validate(buf []byte, signature []byte) bool {
	sum := sha256.Sum256(buf)
	return subtle.ConstantTimeCompare(sum[:], signature) == 1
}
