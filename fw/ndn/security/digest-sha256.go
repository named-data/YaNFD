/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package security

import (
	"bytes"
	"crypto/sha256"
)

// DigestSha256 represents a signer that performs a SHA-256 digest over the packet.
type DigestSha256 struct {
}

// Sign signs a buffer using DigestSha256.
func (d *DigestSha256) Sign(buf []byte) ([]byte, error) {
	sha := sha256.New()
	sha.Write(buf)
	return sha.Sum(nil), nil
}

// Validate returns whether the provided signature is valid for the provided buffer.
func (d *DigestSha256) Validate(buf []byte, signature []byte) bool {
	newSignature, err := d.Sign(buf)
	if err != nil {
		return false
	}
	return bytes.Equal(newSignature, signature)
}
