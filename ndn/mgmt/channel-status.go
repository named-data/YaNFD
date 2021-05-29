/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/tlv"
)

// ChannelStatus contains status information about a channel.
type ChannelStatus struct {
	LocalURI ndn.URI
}

// MakeChannelStatus creates a ChannelStatus.
func MakeChannelStatus(localURI *ndn.URI) *ChannelStatus {
	c := new(ChannelStatus)
	c.LocalURI = *localURI
	return c
}

// Encode encodes a ChannelStatus.
func (c *ChannelStatus) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.ChannelStatus)
	wire.Append(tlv.NewBlock(tlv.LocalURI, []byte(c.LocalURI.String())))
	wire.Encode()
	return wire, nil
}
