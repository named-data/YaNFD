/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/mgmt"
	"github.com/named-data/YaNFD/ndn/tlv"
)

func decodeControlParameters(m Module, interest *ndn.Interest) *mgmt.ControlParameters {
	paramsRaw, _, err := tlv.DecodeBlock(interest.Name().At(m.getManager().prefixLength() + 2).Value())
	if err != nil {
		core.LogWarn(m, "Could not decode ControlParameters in ", interest.Name(), ": ", err)
		return nil
	}
	params, err := mgmt.DecodeControlParameters(paramsRaw)
	if err != nil {
		core.LogWarn(m, "Could not decode ControlParameters in ", interest.Name(), ": ", err)
		return nil
	}
	return params
}
