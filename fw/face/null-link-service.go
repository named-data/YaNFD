/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

// NullLinkService is a link service that drops all packets.
type NullLinkService struct {
	linkServiceBase
}

// MakeNullLinkService makes a NullLinkService.
func MakeNullLinkService(transport transport) *NullLinkService {
	var l NullLinkService
	l.makeLinkServiceBase(transport)
	return &l
}
