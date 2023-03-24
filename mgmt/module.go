/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
)

// Module represents a management module
type Module interface {
	String() string
	registerManager(manager *Thread)
	getManager() *Thread
	handleIncomingInterest(interest *spec.Interest, pitToken []byte, inFace uint64)
}
