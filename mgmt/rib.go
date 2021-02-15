/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"strconv"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
	"github.com/eric135/YaNFD/table"
)

// RIBModule is the module that handles RIB Management.
type RIBModule struct {
	manager               *Thread
	nextRIBDatasetVersion uint64
}

func (r *RIBModule) String() string {
	return "RIBMgmt"
}

func (r *RIBModule) registerManager(manager *Thread) {
	r.manager = manager
}

func (r *RIBModule) getManager() *Thread {
	return r.manager
}

func (r *RIBModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	// Dispatch by verb
	verb := interest.Name().At(r.manager.prefixLength() + 1).String()
	switch verb {
	case "register":
		r.register(interest, pitToken, inFace)
	case "unregister":
		r.unregister(interest, pitToken, inFace)
	case "list":
		r.list(interest, pitToken, inFace)
	default:
		core.LogWarn(r, "Received Interest for non-existent verb '"+verb+"'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (r *RIBModule) register(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < r.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(r, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(r, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name == nil {
		core.LogWarn(r, "Missing Name in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	faceID := inFace
	if params.FaceID != nil && *params.FaceID != 0 {
		faceID = *params.FaceID
		if face.FaceTable.Get(faceID) == nil {
			response = mgmt.MakeControlResponse(410, "Face does not exist", nil)
			r.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}
	}

	origin := table.RouteOriginApp
	if params.Origin != nil {
		origin = *params.Origin
	}

	cost := uint64(0)
	if params.Cost != nil {
		cost = *params.Cost
	}

	flags := table.RouteFlagChildInherit
	if params.Flags != nil {
		flags = *params.Flags
	}

	expirationPeriod := (*time.Duration)(nil)
	if params.ExpirationPeriod != nil {
		expirationPeriod = new(time.Duration)
		*expirationPeriod = time.Duration(*params.ExpirationPeriod) * time.Millisecond
	}

	table.Rib.AddRoute(params.Name, faceID, origin, cost, flags, expirationPeriod)

	if expirationPeriod != nil {
		core.LogInfo(r, "Created route for "+params.Name.String()+" to FaceID="+strconv.FormatUint(faceID, 10)+", Origin="+strconv.FormatUint(origin, 10)+", Cost="+strconv.FormatUint(cost, 10)+", Flags=0x"+strconv.FormatUint(flags, 16)+", ExpirationPeriod="+expirationPeriod.String())
	} else {
		core.LogInfo(r, "Created route for "+params.Name.String()+" to FaceID="+strconv.FormatUint(faceID, 10)+", Origin="+strconv.FormatUint(origin, 10)+", Cost="+strconv.FormatUint(cost, 10)+", Flags=0x"+strconv.FormatUint(flags, 16))
	}
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParams.FaceID = new(uint64)
	*responseParams.FaceID = faceID
	responseParams.Origin = new(uint64)
	*responseParams.Origin = origin
	responseParams.Cost = new(uint64)
	*responseParams.Cost = cost
	responseParams.Flags = new(uint64)
	*responseParams.Flags = flags
	if expirationPeriod != nil {
		responseParams.ExpirationPeriod = new(uint64)
		*responseParams.ExpirationPeriod = uint64(expirationPeriod.Milliseconds())
	}
	responseParamsWire, err := params.Encode()
	if err != nil {
		core.LogError(r, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	r.manager.sendResponse(response, interest, pitToken, inFace)
}

func (r *RIBModule) unregister(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < r.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(r, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(r, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name == nil {
		core.LogWarn(r, "Missing Name in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	faceID := inFace
	if params.FaceID != nil && *params.FaceID != 0 {
		faceID = *params.FaceID
	}

	origin := table.RouteOriginApp
	if params.Origin != nil {
		origin = *params.Origin
	}

	table.Rib.RemoveRoute(params.Name, faceID, origin)

	core.LogInfo(r, "Removed route for "+params.Name.String()+", FaceID="+strconv.FormatUint(faceID, 10)+", Origin="+strconv.FormatUint(origin, 10))
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParams.FaceID = new(uint64)
	*responseParams.FaceID = faceID
	responseParams.Origin = new(uint64)
	*responseParams.Origin = origin
	responseParamsWire, err := params.Encode()
	if err != nil {
		core.LogError(r, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	r.manager.sendResponse(response, interest, pitToken, inFace)
}

func (r *RIBModule) list(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() > r.manager.prefixLength()+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	entries := table.Rib.GetAllEntries()
	dataset := make([]byte, 0)
	for _, entry := range entries {
		ribEntry := mgmt.MakeRibEntry(entry.Name)
		for _, route := range entry.GetRoutes() {
			var res mgmt.Route
			res.FaceID = route.FaceID
			res.Origin = route.Origin
			res.Cost = route.Cost
			res.Flags = route.Flags
			if route.ExpirationPeriod != nil {
				res.ExpirationPeriod = route.ExpirationPeriod
			}
			ribEntry.Routes = append(ribEntry.Routes, res)
		}

		wire, err := ribEntry.Encode()
		if err != nil {
			core.LogError(r, "Cannot encode RibEntry for Name="+entry.Name.String()+": "+err.Error())
			continue
		}
		encoded, err := wire.Wire()
		if err != nil {
			core.LogError(r, "Cannot encode RibEntry for Name="+entry.Name.String()+": "+err.Error())
			continue
		}
		dataset = append(dataset, encoded...)
	}

	name, _ := ndn.NameFromString(r.manager.prefix.String() + "/rib/list")
	segments := mgmt.MakeStatusDataset(name, r.nextRIBDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(r, "Unable to encode RIB dataset: "+err.Error())
			return
		}
		r.manager.transport.Send(encoded, []byte{}, nil)
	}

	core.LogTrace(r, "Published RIB dataset version="+strconv.FormatUint(r.nextRIBDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	r.nextRIBDatasetVersion++
}
