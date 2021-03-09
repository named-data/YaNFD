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
	"github.com/eric135/YaNFD/face"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
	"github.com/eric135/YaNFD/table"
)

// FIBModule is the module that handles FIB Management.
type FIBModule struct {
	manager               *Thread
	nextFIBDatasetVersion uint64
}

func (f *FIBModule) String() string {
	return "FIBMgmt"
}

func (f *FIBModule) registerManager(manager *Thread) {
	f.manager = manager
}

func (f *FIBModule) getManager() *Thread {
	return f.manager
}

func (f *FIBModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	// Dispatch by verb
	verb := interest.Name().At(f.manager.prefixLength() + 1).String()
	switch verb {
	case "add-nexthop":
		f.add(interest, pitToken, inFace)
	case "remove-nexthop":
		f.remove(interest, pitToken, inFace)
	case "list":
		f.list(interest, pitToken, inFace)
	default:
		core.LogWarn(f, "Received Interest for non-existent verb '"+verb+"'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (f *FIBModule) add(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(f, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name == nil {
		core.LogWarn(f, "Missing Name in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	faceID := inFace
	if params.FaceID != nil && *params.FaceID != 0 {
		faceID = *params.FaceID
		if face.FaceTable.Get(faceID) == nil {
			response = mgmt.MakeControlResponse(410, "Face does not exist", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}
	}

	cost := uint64(0)
	if params.Cost != nil {
		cost = *params.Cost
	}

	table.FibStrategyTable.AddNexthop(params.Name, faceID, cost)

	core.LogInfo(f, "Created nexthop for "+params.Name.String()+" to FaceID="+strconv.FormatUint(faceID, 10)+"with Cost="+strconv.FormatUint(cost, 10))
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParams.FaceID = new(uint64)
	*responseParams.FaceID = faceID
	responseParams.Cost = new(uint64)
	*responseParams.Cost = cost
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FIBModule) remove(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(f, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name == nil {
		core.LogWarn(f, "Missing Name in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	faceID := inFace
	if params.FaceID != nil && *params.FaceID != 0 {
		faceID = *params.FaceID
	}

	table.FibStrategyTable.RemoveNexthop(params.Name, faceID)

	core.LogInfo(f, "Removed nexthop for "+params.Name.String()+" to FaceID="+strconv.FormatUint(faceID, 10))
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParams.FaceID = new(uint64)
	*responseParams.FaceID = faceID
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FIBModule) list(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() > f.manager.prefixLength()+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	// TODO: For thread safety, we should lock the FIB from writes until we are done
	entries := table.FibStrategyTable.GetAllFIBEntries()
	dataset := make([]byte, 0)
	for _, fsEntry := range entries {
		fibEntry := mgmt.MakeFibEntry(fsEntry.Name)
		for _, nexthop := range fsEntry.GetNexthops() {
			var record mgmt.NextHopRecord
			record.FaceID = nexthop.Nexthop
			record.Cost = nexthop.Cost
			fibEntry.Nexthops = append(fibEntry.Nexthops, record)
		}

		wire, err := fibEntry.Encode()
		if err != nil {
			core.LogError(f, "Cannot encode FibEntry for Name="+fsEntry.Name.String()+": "+err.Error())
			continue
		}
		encoded, err := wire.Wire()
		if err != nil {
			core.LogError(f, "Cannot encode FibEntry for Name="+fsEntry.Name.String()+": "+err.Error())
			continue
		}
		dataset = append(dataset, encoded...)
	}

	name, _ := ndn.NameFromString(f.manager.prefix.String() + "/fib/list")
	segments := mgmt.MakeStatusDataset(name, f.nextFIBDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode FIB dataset: "+err.Error())
			return
		}
		f.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(f, "Published FIB dataset version="+strconv.FormatUint(f.nextFIBDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	f.nextFIBDatasetVersion++
}
