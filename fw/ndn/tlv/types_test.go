/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package tlv_test

import (
	"testing"

	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/stretchr/testify/assert"
)

func TestCritical(t *testing.T) {
	assert.True(t, tlv.IsCritical(0x01))
	assert.True(t, tlv.IsCritical(0x1F))
	assert.False(t, tlv.IsCritical(0x20))
	assert.True(t, tlv.IsCritical(0x21))
	assert.False(t, tlv.IsCritical(0x2000))
	assert.True(t, tlv.IsCritical(0x2001))
}
