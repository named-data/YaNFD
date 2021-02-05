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
)

// FaceModule is the module that handles for Face Management.
type FaceModule struct {
	manager *Thread
}

func (f *FaceModule) String() string {
	return "FaceMgmt"
}

func (f *FaceModule) registerManager(manager *Thread) {
	f.manager = manager
}

func (f *FaceModule) getManager() *Thread {
	return f.manager
}

func (f *FaceModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace int) {
	// Dispatch by verb
	verb := interest.Name().At(f.manager.prefixLength() + 1).String()
	switch verb {
	case "create":
		f.create(interest, pitToken, inFace)
	case "update":
		f.update(interest, pitToken, inFace)
	case "destroy":
		f.destroy(interest, pitToken, inFace)
	case "list":
		f.list(interest, pitToken, inFace)
	case "query":
		f.query(interest, pitToken, inFace)
	case "channels":
		f.channels(interest, pitToken, inFace)
	default:
		core.LogWarn(f, "Received Interest for non-existent verb '"+verb+"'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (f *FaceModule) create(interest *ndn.Interest, pitToken []byte, inFace int) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < 5 {
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

	if params.URI == nil {
		core.LogWarn(f, "Missing URI in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.URI.Canonize() != nil {
		core.LogWarn(f, "Cannot canonize remote URI in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(406, "URI could not be canonized", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Ensure does not conflict with existing face
	existingFace := face.FaceTable.GetByURI(params.URI)
	if existingFace != nil {
		core.LogWarn(f, "Cannot create face "+params.URI.String()+": Conflicts with existing face FaceID="+strconv.Itoa(existingFace.FaceID())+", RemoteURI="+existingFace.RemoteURI().String())
		responseParams := mgmt.MakeControlParameters()
		f.fillFaceProperties(responseParams, existingFace)
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode response parameters: "+err.Error())
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(409, "Conflicts with existing face", responseParamsWire)
		}
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	var linkService face.LinkService

	if params.URI.Scheme() == "udp4" || params.URI.Scheme() == "udp6" {
		// Check if multicast address
		// TODO

		// Validate and populate missing optional params
		if params.LocalURI != nil {
			if params.LocalURI.Canonize() != nil {
				core.LogWarn(f, "Cannot canonize local URI in ControlParameters for "+interest.Name().String())
				response = mgmt.MakeControlResponse(406, "LocalURI could not be canonized", nil)
				return
			}
			if params.LocalURI.Scheme() != params.URI.Scheme() {
				core.LogWarn(f, "Local URI scheme does not match remote URI scheme in ControlParameters for "+interest.Name().String())
				response = mgmt.MakeControlResponse(406, "LocalURI scheme does not match URI scheme", nil)
			}
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		// FacePersistency, BaseCongestionMarkingInterval, DefaultCongestionThreshold, Mtu, and Flags are ignored for now

		// Create new UDP face
		transport, err := face.MakeUnicastUDPTransport(params.URI, params.LocalURI)
		if err != nil {
			core.LogWarn(f, "Unable to create unicast UDP face with URI "+params.URI.String()+": Unsupported scheme "+params.URI.Scheme())
			response = mgmt.MakeControlResponse(406, "Unsupported scheme "+params.URI.Scheme(), nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}
		linkService = face.MakeNDNLPLinkService(transport, face.NDNLPLinkServiceOptions{})
		face.FaceTable.Add(linkService)

		// Start new face
		go linkService.Run()
	} else {
		// Unsupported scheme
		core.LogWarn(f, "Cannot create face with URI "+params.URI.String()+": Unsupported scheme "+params.URI.Scheme())
		response = mgmt.MakeControlResponse(406, "Unsupported scheme "+params.URI.Scheme(), nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if linkService == nil {
		// Internal failure --> 504
		core.LogWarn(f, "Transport error when creating face "+params.URI.String())
		response = mgmt.MakeControlResponse(504, "Transport error when creating face", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	core.LogInfo(f, "Created face with URI "+params.URI.String())
	responseParams := mgmt.MakeControlParameters()
	f.fillFaceProperties(responseParams, linkService)
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "Face created", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
	return
}

func (f *FaceModule) update(interest *ndn.Interest, pitToken []byte, inFace int) {
	// TODO
}

func (f *FaceModule) destroy(interest *ndn.Interest, pitToken []byte, inFace int) {
	// TODO
}

func (f *FaceModule) list(interest *ndn.Interest, pitToken []byte, inFace int) {
	// TODO
}

func (f *FaceModule) query(interest *ndn.Interest, pitToken []byte, inFace int) {
	// TODO
}

func (f *FaceModule) channels(interest *ndn.Interest, pitToken []byte, inFace int) {
	// We don't have channels in YaNFD, so just return an empty list
	// TODO
}

func (f *FaceModule) fillFaceProperties(params *mgmt.ControlParameters, face face.LinkService) {
	params.FaceID = new(uint64)
	*params.FaceID = uint64(face.FaceID())
	params.URI = face.RemoteURI()
	params.LocalURI = face.LocalURI()
	// TODO: Face Persistency
	params.FacePersistency = new(uint64)
	*params.FacePersistency = 0
	params.MTU = new(uint64)
	*params.MTU = uint64(face.MTU())
	// TODO: Flags
	params.Flags = new(uint64)
	*params.Flags = 0
}
