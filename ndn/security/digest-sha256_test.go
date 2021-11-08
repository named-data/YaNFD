/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package security_test

import (
	"crypto/subtle"
	"encoding/hex"
	"testing"

	"github.com/named-data/YaNFD/ndn/security"
	"github.com/stretchr/testify/assert"
)

func TestDigestSha256Sign(t *testing.T) {
	// https://www.di-mgt.com.au/sha_testvectors.html
	buf := []byte("abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq")
	ref, _ := hex.DecodeString("248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1")

	var signer security.DigestSha256
	sig, e := signer.Sign(buf)
	assert.NoError(t, e)
	assert.Equal(t, 1, subtle.ConstantTimeCompare(sig, ref))
}

func TestDigestSha256Verify(t *testing.T) {
	// https://www.di-mgt.com.au/sha_testvectors.html
	buf := []byte{}
	ref, _ := hex.DecodeString("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	wrongA := ref[1:]
	wrongB := append([]byte{0x00}, ref...)
	wrongC := append([]byte{}, ref...)
	wrongC[4] ^= 0x01

	var signer security.DigestSha256
	assert.True(t, signer.Validate(buf, ref))
	assert.False(t, signer.Validate(buf, wrongA))
	assert.False(t, signer.Validate(buf, wrongB))
	assert.False(t, signer.Validate(buf, wrongC))
}
