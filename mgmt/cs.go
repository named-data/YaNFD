/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
	"github.com/eric135/YaNFD/table"
)

// ContentStoreModule is the module that handles Content Store Management.
type ContentStoreModule struct {
	manager *Thread
	//nextCsDatasetVersion uint64
	csPrefix *ndn.Name
}

func (c *ContentStoreModule) String() string {
	return "ContentStoreMgmt"
}

func (c *ContentStoreModule) registerManager(manager *Thread) {
	c.manager = manager
	c.csPrefix = c.manager.localPrefix.DeepCopy().Append(ndn.NewGenericNameComponent([]byte("cs")))
}

func (c *ContentStoreModule) getManager() *Thread {
	return c.manager
}

func (c *ContentStoreModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	// Only allow from /localhost
	if !c.manager.localPrefix.PrefixOf(interest.Name()) {
		core.LogWarn(c, "Received CS management Interest from non-local source - DROP")
		return
	}

	// Dispatch by verb
	verb := interest.Name().At(c.manager.prefixLength() + 1).String()
	switch verb {
	case "config":
		c.config(interest, pitToken, inFace)
	case "erase":
		// TODO
		//c.erase(interest, pitToken, inFace)
	case "info":
		// TODO
		//c.info(interest, pitToken, inFace)
	case "query":
		// TODO
		//c.query(interest, pitToken, inFace)
	default:
		core.LogWarn(c, "Received Interest for non-existent verb '"+verb+"'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		c.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (c *ContentStoreModule) config(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < c.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(c, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		c.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(c, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		c.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if (params.Flags == nil && params.Mask != nil) || (params.Flags != nil && params.Mask == nil) {
		core.LogWarn(c, "Flags and Mask fields must either both be present or both be not present")
		response = mgmt.MakeControlResponse(409, "ControlParameters are incorrect", nil)
		c.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Capacity != nil {
		core.LogInfo(c, "Setting CS capacity to "+strconv.FormatUint(*params.Capacity, 10))
		table.SetCsCapacity(int(*params.Capacity))
	}

	/*if params.Flags != nil {
		if *params.Mask&0x01 > 0 {
			// CS_ENABLE_ADMIT
			// TODO
		}

		if *params.Mask&0x02 > 0 {
			// CS_ENABLE_SERVE
			// TODO
		}
	}*/

	responseParams := mgmt.MakeControlParameters()
	responseParams.Capacity = params.Capacity
	responseParams.Flags = new(uint64)
	*responseParams.Flags = 0
	// TODO: *responseParams.Flags += 1 if CS_ENABLE_ADMIT
	// TODO: *responseParams.Flags += 2 if CS_ENABLE_SERVE
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(c, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	c.manager.sendResponse(response, interest, pitToken, inFace)
}
