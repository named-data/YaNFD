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

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/mgmt"
	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
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
	case "announce":
		r.announce(interest, pitToken, inFace)
	case "list":
		r.list(interest, pitToken, inFace)
	default:
		core.LogWarn(r, "Received Interest for non-existent verb '", verb, "'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (r *RIBModule) register(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < r.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(r, "Missing ControlParameters in ", interest.Name())
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
		core.LogWarn(r, "Missing Name in ControlParameters for ", interest.Name())
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

	convert, _ := enc.NameFromStr(params.Name.String())
	table.Rib.AddEncRoute(&convert, faceID, origin, cost, flags, expirationPeriod)
	if expirationPeriod != nil {
		core.LogInfo(r, "Created route for Prefix=", params.Name, ", FaceID=", faceID, ", Origin=", origin, ", Cost=", cost, ", Flags=0x", strconv.FormatUint(flags, 16), ", ExpirationPeriod=", expirationPeriod)
	} else {
		core.LogInfo(r, "Created route for Prefix=", params.Name, ", FaceID=", faceID, ", Origin=", origin, ", Cost=", cost, ", Flags=0x", strconv.FormatUint(flags, 16))
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
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(r, "Unable to encode response parameters: ", err)
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
		core.LogWarn(r, "Missing ControlParameters in ", interest.Name())
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
		core.LogWarn(r, "Missing Name in ControlParameters for ", interest.Name())
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
	convert, _ := enc.NameFromStr(params.Name.String())
	table.Rib.RemoveRouteEnc(&convert, faceID, origin)

	core.LogInfo(r, "Removed route for Prefix=", params.Name, ", FaceID=", faceID, ", Origin=", origin)
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParams.FaceID = new(uint64)
	*responseParams.FaceID = faceID
	responseParams.Origin = new(uint64)
	*responseParams.Origin = origin
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(r, "Unable to encode response parameters: ", err)
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	r.manager.sendResponse(response, interest, pitToken, inFace)
}

func (r *RIBModule) announce(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse
	if interest.Name().Size() != r.manager.prefixLength()+3 || interest.Name().At(r.manager.prefixLength()+2).Type() != tlv.ParametersSha256DigestComponent {
		// Name not long enough to contain ControlParameters
		core.LogWarn(r, "Name of Interest=", interest.Name(), " is either too short or incorrectly formatted to be rib/announce")
		response = mgmt.MakeControlResponse(400, "Name is incorrect", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Get PrefixAnnouncement
	if len(interest.ApplicationParameters()) == 0 || interest.ApplicationParameters()[0].Type() != tlv.Data {
		core.LogWarn(r, "PrefixAnnouncement Interest=", interest.Name(), " missing PrefixAnnouncement")
		response = mgmt.MakeControlResponse(400, "PrefixAnnouncement is missing", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	prefixAnnouncement, err := ndn.DecodePrefixAnnouncement(interest.ApplicationParameters()[0].DeepCopy())
	if err != nil {
		core.LogWarn(r, "PrefixAnnouncement Interest=", interest.Name(), " has invalid PrefixAnnouncement")
		response = mgmt.MakeControlResponse(400, "PrefixAnnouncement is invalid", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	prefix := prefixAnnouncement.Prefix()
	faceID := inFace
	origin := table.RouteOriginPrefixAnn
	cost := uint64(0)
	expirationPeriod := time.Duration(prefixAnnouncement.ExpirationPeriod()) * time.Millisecond

	// Use more restrictive of ExpirationPeriod and ValidityPeriod
	notBefore, notAfter := prefixAnnouncement.ValidityPeriod()
	if notBefore.Unix() == 0 && notAfter.Unix() == 0 {
	} else if notBefore.After(time.Now()) && notAfter.Before(time.Now()) {
		core.LogWarn(r, "PrefixAnnouncement Interest=", interest.Name(), " is in the future")
		response = mgmt.MakeControlResponse(416, "Time out of range", nil)
		r.manager.sendResponse(response, interest, pitToken, inFace)
		return
	} else if notAfter.Before(time.Now().Add(expirationPeriod)) {
		expirationPeriod = time.Until(notAfter)
	}
	convert, _ := enc.NameFromStr(prefix.String())
	table.Rib.AddEncRoute(&convert, faceID, origin, cost, 0, &expirationPeriod)

	core.LogInfo(r, "Created route via PrefixAnnouncement for Prefix=", prefix, ", FaceID=", faceID, ", Origin=", origin, ", Cost=", cost, ", Flags=0x0, ExpirationPeriod=", expirationPeriod)

	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = prefix
	responseParams.FaceID = new(uint64)
	*responseParams.FaceID = faceID
	responseParams.Origin = new(uint64)
	*responseParams.Origin = origin
	responseParams.Cost = new(uint64)
	*responseParams.Cost = cost
	responseParams.Flags = new(uint64)
	*responseParams.Flags = 0
	responseParams.ExpirationPeriod = new(uint64)
	*responseParams.ExpirationPeriod = uint64(expirationPeriod.Milliseconds())
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(r, "Unable to encode response parameters: ", err)
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
		convert, _ := ndn.NameFromString(entry.EncName.String())
		ribEntry := mgmt.MakeRibEntry(convert)
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
			core.LogError(r, "Cannot encode RibEntry for Name=", entry.Name, ": ", err)
			continue
		}
		encoded, err := wire.Wire()
		if err != nil {
			core.LogError(r, "Cannot encode RibEntry for Name=", entry.Name, ": ", err)
			continue
		}
		dataset = append(dataset, encoded...)
	}

	name, _ := ndn.NameFromString(interest.Name().Prefix(r.manager.prefixLength()).String() + "/rib/list")
	segments := mgmt.MakeStatusDataset(name, r.nextRIBDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(r, "Unable to encode RIB dataset: ", err)
			return
		}
		r.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(r, "Published RIB dataset version=", r.nextRIBDatasetVersion, ", containing ", len(segments), " segments")
	r.nextRIBDatasetVersion++
}
