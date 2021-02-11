/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"net"
	"sort"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
)

// FaceModule is the module that handles for Face Management.
type FaceModule struct {
	manager                *Thread
	nextFaceDatasetVersion uint64
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
		// Check that remote endpoint is not a unicast address
		if remoteAddr := net.ParseIP(params.URI.Path()); remoteAddr != nil && !remoteAddr.IsGlobalUnicast() && !remoteAddr.IsLinkLocalUnicast() {
			core.LogWarn(f, "Cannot create unicast UDP face to non-unicast address "+params.URI.String())
			response = mgmt.MakeControlResponse(406, "URI must be unicast", nil)
		}

		// Validate and populate missing optional params
		// TODO: Validate and use LocalURI if present
		/*if params.LocalURI != nil {
			if params.LocalURI.Canonize() != nil {
				core.LogWarn(f, "Cannot canonize local URI in ControlParameters for "+interest.Name().String())
				response = mgmt.MakeControlResponse(406, "LocalURI could not be canonized", nil)
				return
			}
			if params.LocalURI.Scheme() != params.URI.Scheme() {
				core.LogWarn(f, "Local URI scheme does not match remote URI scheme in ControlParameters for "+interest.Name().String())
				response = mgmt.MakeControlResponse(406, "LocalURI scheme does not match URI scheme", nil)
				f.manager.sendResponse(response, interest, pitToken, inFace)
				return
			}
			// TODO: Check if matches a local interface IP
		}*/

		// FacePersistency, BaseCongestionMarkingInterval, DefaultCongestionThreshold, Mtu, and Flags are ignored for now

		// Create new UDP face
		transport, err := face.MakeUnicastUDPTransport(params.URI, nil)
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
	if interest.Name().Size() > f.manager.prefixLength()+2 {
		// Ignore because contains version and/or segment components
		return
	}

	dataset := make([]byte, 0)

	// Generate new dataset
	faces := make(map[int]face.LinkService)
	faceIDs := make([]int, 0)
	for _, face := range face.FaceTable.GetAll() {
		faces[face.FaceID()] = face
		faceIDs = append(faceIDs, face.FaceID())
	}
	// We have to sort these or they appear in a strange order
	sort.Sort(sort.IntSlice(faceIDs))
	for _, faceID := range faceIDs {
		dataset = append(dataset, f.createDataset(faces[faceID])...)
	}

	name, _ := ndn.NameFromString(f.manager.prefix.String() + "/faces/list")
	segments := mgmt.MakeStatusDataset(name, f.nextFaceDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to enable face status dataset: "+err.Error())
		}
		f.manager.transport.Send(encoded, []byte{}, nil)
	}

	core.LogTrace(f, "Published face dataset version="+strconv.FormatUint(f.nextFaceDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	f.nextFaceDatasetVersion++
}

func (f *FaceModule) query(interest *ndn.Interest, pitToken []byte, inFace int) {
	// TODO
}

func (f *FaceModule) createDataset(face face.LinkService) []byte {
	faceDataset := mgmt.MakeFaceStatus()
	faceDataset.FaceID = uint64(face.FaceID())
	faceDataset.URI = face.RemoteURI()
	faceDataset.LocalURI = face.LocalURI()
	// TODO: ExpirationPeriod
	faceDataset.FaceScope = uint64(face.Scope())
	// TODO: Put a real value here
	faceDataset.FacePersistency = 0
	faceDataset.LinkType = uint64(face.LinkType())
	// TODO: BaseCongestionMarkingInterval
	// TODO: DefaultCongestionThreshold
	faceDataset.MTU = new(uint64)
	*faceDataset.MTU = uint64(face.MTU())
	// TODO: Put real values here
	faceDataset.NInInterests = 0
	faceDataset.NInData = 0
	faceDataset.NInNacks = 0
	faceDataset.NOutInterests = 0
	faceDataset.NOutData = 0
	faceDataset.NOutNacks = 0
	faceDataset.NInBytes = 0
	faceDataset.NOutBytes = 0
	// TODO: Put a real value here
	faceDataset.Flags = 0

	faceDatasetEncoded, err := faceDataset.Encode()
	if err != nil {
		core.LogError(f, "Cannot encode FaceStatus for FaceID="+strconv.Itoa(face.FaceID())+": "+err.Error())
		return []byte{}
	}
	faceDatasetWire, err := faceDatasetEncoded.Wire()
	if err != nil {
		core.LogError(f, "Cannot encode FaceStatus for FaceID="+strconv.Itoa(face.FaceID())+": "+err.Error())
		return []byte{}
	}
	return faceDatasetWire
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
