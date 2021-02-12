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
	manager                   *Thread
	nextFaceDatasetVersion    uint64
	nextChannelDatasetVersion uint64
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

func (f *FaceModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace uint64) {
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

func (f *FaceModule) create(interest *ndn.Interest, pitToken []byte, inFace uint64) {
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
		core.LogWarn(f, "Cannot create face "+params.URI.String()+": Conflicts with existing face FaceID="+strconv.FormatUint(existingFace.FaceID(), 10)+", RemoteURI="+existingFace.RemoteURI().String())
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
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) update(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	// TODO
}

func (f *FaceModule) destroy(interest *ndn.Interest, pitToken []byte, inFace uint64) {
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

	if params.FaceID == nil {
		core.LogWarn(f, "Missing FaceId in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if face.FaceTable.Get(*params.FaceID) != nil {
		face.FaceTable.Remove(*params.FaceID)
		core.LogInfo(f, "Destroyed face with FaceID="+strconv.FormatUint(*params.FaceID, 10))
	} else {
		core.LogInfo(f, "Ignoring attempt to delete non-existent face with FaceID="+strconv.FormatUint(*params.FaceID, 10))
	}

	responseParamsWire, err := params.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) list(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() > f.manager.prefixLength()+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	faces := make(map[uint64]face.LinkService)
	faceIDs := make([]uint64, 0)
	for _, face := range face.FaceTable.GetAll() {
		faces[face.FaceID()] = face
		faceIDs = append(faceIDs, face.FaceID())
	}
	// We have to sort these or they appear in a strange order
	sort.Slice(faceIDs, func(a int, b int) bool { return faceIDs[a] < faceIDs[b] })
	dataset := make([]byte, 0)
	for _, faceID := range faceIDs {
		dataset = append(dataset, f.createDataset(faces[faceID])...)
	}

	name, _ := ndn.NameFromString(f.manager.prefix.String() + "/faces/list")
	segments := mgmt.MakeStatusDataset(name, f.nextFaceDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode face status dataset: "+err.Error())
			return
		}
		f.manager.transport.Send(encoded, []byte{}, nil)
	}

	core.LogTrace(f, "Published face dataset version="+strconv.FormatUint(f.nextFaceDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	f.nextFaceDatasetVersion++
}

func (f *FaceModule) query(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain FaceQueryFilter
		core.LogWarn(f, "Missing FaceQueryFilter in "+interest.Name().String())
		return
	}

	filter, err := mgmt.DecodeFaceQueryFilterFromEncoded(interest.Name().At(f.manager.prefixLength() + 2).Value())
	if err != nil {
		return
	}

	faces := face.FaceTable.GetAll()
	matchingFaces := make([]int, 0)
	for pos, face := range faces {
		if filter.FaceID != nil && *filter.FaceID != face.FaceID() {
			continue
		}

		if filter.URIScheme != nil && *filter.URIScheme != face.LocalURI().Scheme() && *filter.URIScheme != face.RemoteURI().Scheme() {
			continue
		}

		if filter.URI != nil && filter.URI.String() != face.RemoteURI().String() {
			continue
		}

		if filter.LocalURI != nil && filter.LocalURI.String() != face.LocalURI().String() {
			continue
		}

		if filter.FaceScope != nil && *filter.FaceScope != uint64(face.Scope()) {
			continue
		}

		// TODO: Add FacePersistency to Face
		/*if filter.FacePersistency != nil && *filter.FacePersistency != uint64(face.Persistency) {
			continue
		}*/

		if filter.LinkType != nil && *filter.LinkType != uint64(face.LinkType()) {
			continue
		}

		matchingFaces = append(matchingFaces, pos)
	}

	// We have to sort these or they appear in a strange order
	//sort.Slice(matchingFaces, func(a int, b int) bool { return matchingFaces[a] < matchingFaces[b] })

	dataset := make([]byte, 0)
	for _, pos := range matchingFaces {
		dataset = append(dataset, f.createDataset(faces[pos])...)
	}

	segments := mgmt.MakeStatusDataset(interest.Name(), f.nextFaceDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode face query dataset: "+err.Error())
			return
		}
		f.manager.transport.Send(encoded, []byte{}, nil)
	}

	core.LogTrace(f, "Published face query dataset version="+strconv.FormatUint(f.nextFaceDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	f.nextFaceDatasetVersion++
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
		core.LogError(f, "Cannot encode FaceStatus for FaceID="+strconv.FormatUint(face.FaceID(), 10)+": "+err.Error())
		return []byte{}
	}
	faceDatasetWire, err := faceDatasetEncoded.Wire()
	if err != nil {
		core.LogError(f, "Cannot encode FaceStatus for FaceID="+strconv.FormatUint(face.FaceID(), 10)+": "+err.Error())
		return []byte{}
	}
	return faceDatasetWire
}

func (f *FaceModule) channels(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() < f.manager.prefixLength()+2 {
		core.LogWarn(f, "Channel dataset Interest too short: "+interest.Name().String())
		return
	}

	dataset := make([]byte, 0)
	// UDP channel
	ifaces, err := net.Interfaces()
	if err != nil {
		core.LogWarn(f, "Unable to access channel dataset: "+err.Error())
		return
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			core.LogWarn(f, "Unable to access IP addresses for "+iface.Name+": "+err.Error())
			return
		}
		for _, addr := range addrs {
			ipAddr := addr.(*net.IPNet)

			ipVersion := 4
			path := ipAddr.IP.String()
			if ipAddr.IP.To4() == nil {
				ipVersion = 6
				path += "%" + iface.Name
			}

			if !addr.(*net.IPNet).IP.IsLoopback() {
				uri := ndn.MakeUDPFaceURI(ipVersion, path, face.NDNUnicastUDPPort)
				channel := mgmt.MakeChannelStatus(uri)
				channelEncoded, err := channel.Encode()
				if err != nil {
					core.LogError(f, "Cannot encode ChannelStatus for Channel="+uri.String()+": "+err.Error())
					continue
				}
				channelWire, err := channelEncoded.Wire()
				if err != nil {
					core.LogError(f, "Cannot encode ChannelStatus for Channel="+uri.String()+": "+err.Error())
					continue
				}
				dataset = append(dataset, channelWire...)
			}
		}
	}

	// Unix channel
	uri := ndn.MakeUnixFaceURI(face.NDNUnixSocketFile)
	channel := mgmt.MakeChannelStatus(uri)
	channelEncoded, err := channel.Encode()
	if err != nil {
		core.LogError(f, "Cannot encode ChannelStatus for Channel="+uri.String()+": "+err.Error())
		return
	}
	channelWire, err := channelEncoded.Wire()
	if err != nil {
		core.LogError(f, "Cannot encode ChannelStatus for Channel="+uri.String()+": "+err.Error())
		return
	}
	dataset = append(dataset, channelWire...)

	segments := mgmt.MakeStatusDataset(interest.Name(), f.nextChannelDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode channel dataset: "+err.Error())
			return
		}
		f.manager.transport.Send(encoded, []byte{}, nil)
	}

	core.LogTrace(f, "Published channel dataset version="+strconv.FormatUint(f.nextChannelDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	f.nextChannelDatasetVersion++
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
