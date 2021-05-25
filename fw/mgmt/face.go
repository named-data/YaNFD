/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"math"
	"net"
	"sort"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/face"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// FaceModule is the module that handles Face Management.
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
	// Only allow from /localhost
	if !f.manager.localPrefix.PrefixOf(interest.Name()) {
		core.LogWarn(f, "Received face management Interest from non-local source - DROP")
		return
	}

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
		core.LogWarn(f, "Received Interest for non-existent verb '", verb, "'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (f *FaceModule) create(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in ", interest.Name())
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
		core.LogWarn(f, "Missing URI in ControlParameters for ", interest.Name())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.URI.Canonize() != nil {
		core.LogWarn(f, "Cannot canonize remote URI in ControlParameters for ", interest.Name())
		response = mgmt.MakeControlResponse(406, "URI could not be canonized", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if (params.Flags != nil && params.Mask == nil) || (params.Flags == nil && params.Mask != nil) {
		core.LogWarn(f, "Flags and Mask fields either both be present or both be not present")
		response = mgmt.MakeControlResponse(409, "Incomplete Flags/Mask combination", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Ensure does not conflict with existing face
	existingFace := face.FaceTable.GetByURI(params.URI)
	if existingFace != nil {
		core.LogWarn(f, "Cannot create face ", params.URI, ": Conflicts with existing face FaceID=", existingFace.FaceID(), ", RemoteURI=", existingFace.RemoteURI())
		responseParams := mgmt.MakeControlParameters()
		f.fillFaceProperties(responseParams, existingFace)
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode response parameters: ", err)
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(409, "Conflicts with existing face", responseParamsWire)
		}
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	var linkService *face.NDNLPLinkService

	if params.URI.Scheme() == "udp4" || params.URI.Scheme() == "udp6" {
		// Check that remote endpoint is not a unicast address
		if remoteAddr := net.ParseIP(params.URI.Path()); remoteAddr != nil && !remoteAddr.IsGlobalUnicast() && !remoteAddr.IsLinkLocalUnicast() {
			core.LogWarn(f, "Cannot create unicast UDP face to non-unicast address ", params.URI)
			response = mgmt.MakeControlResponse(406, "URI must be unicast", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		// Validate and populate missing optional params
		// TODO: Validate and use LocalURI if present
		/*if params.LocalURI != nil {
			if params.LocalURI.Canonize() != nil {
				core.LogWarn(f, "Cannot canonize local URI in ControlParameters for ", interest.Name())
				response = mgmt.MakeControlResponse(406, "LocalURI could not be canonized", nil)
				return
			}
			if params.LocalURI.Scheme() != params.URI.Scheme() {
				core.LogWarn(f, "Local URI scheme does not match remote URI scheme in ControlParameters for ", interest.Name())
				response = mgmt.MakeControlResponse(406, "LocalURI scheme does not match URI scheme", nil)
				f.manager.sendResponse(response, interest, pitToken, inFace)
				return
			}
			// TODO: Check if matches a local interface IP
		}*/

		persistency := face.PersistencyPersistent
		if params.FacePersistency != nil && (*params.FacePersistency == uint64(face.PersistencyPersistent) || *params.FacePersistency == uint64(face.PersistencyPermanent)) {
			persistency = face.Persistency(*params.FacePersistency)
		} else if params.FacePersistency != nil {
			core.LogWarn(f, "Unacceptable persistency ", face.Persistency(*params.FacePersistency), " for UDP face specified in ControlParameters for ", interest.Name())
			response = mgmt.MakeControlResponse(406, "Unacceptable persistency", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		baseCongestionMarkingInterval := 100 * time.Millisecond
		if params.BaseCongestionMarkingInterval != nil {
			baseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkingInterval) * time.Nanosecond
		}

		defaultCongestionThresholdBytes := uint64(math.Pow(2, 16))
		if params.DefaultCongestionThreshold != nil {
			defaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		}

		// Create new UDP face
		transport, err := face.MakeUnicastUDPTransport(params.URI, nil, persistency)
		if err != nil {
			core.LogWarn(f, "Unable to create unicast UDP face with URI ", params.URI, ":", err.Error())
			response = mgmt.MakeControlResponse(406, "Transport error", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		if params.MTU != nil {
			mtu := int(*params.MTU)
			if *params.MTU > tlv.MaxNDNPacketSize {
				mtu = tlv.MaxNDNPacketSize
			}
			transport.SetMTU(mtu)
		}

		// NDNLP link service parameters
		options := face.MakeNDNLPLinkServiceOptions()
		if params.Flags != nil {
			// Mask already guaranteed to be present if Flags is above
			flags := *params.Flags
			mask := *params.Mask

			if mask&0x1 == 1 {
				// LocalFieldsEnabled
				if flags&0x1 == 1 {
					options.IsConsumerControlledForwardingEnabled = true
					options.IsIncomingFaceIndicationEnabled = true
					options.IsLocalCachePolicyEnabled = true
				} else {
					options.IsConsumerControlledForwardingEnabled = false
					options.IsIncomingFaceIndicationEnabled = false
					options.IsLocalCachePolicyEnabled = false
				}
			}

			if mask>>1&0x1 == 1 {
				// LpReliabilityEnabled
				options.IsReliabilityEnabled = flags>>1&0x01 == 1
			}

			// Congestion control
			if mask>>2&0x01 == 1 {
				// CongestionMarkingEnabled
				options.IsCongestionMarkingEnabled = flags>>2&0x01 == 1
			}
			options.BaseCongestionMarkingInterval = baseCongestionMarkingInterval
			options.DefaultCongestionThresholdBytes = defaultCongestionThresholdBytes
		}

		linkService = face.MakeNDNLPLinkService(transport, options)
		face.FaceTable.Add(linkService)

		// Start new face
		go linkService.Run()
	} else {
		// Unsupported scheme
		core.LogWarn(f, "Cannot create face with URI ", params.URI, ": Unsupported scheme ", params.URI)
		response = mgmt.MakeControlResponse(406, "Unsupported scheme "+params.URI.Scheme(), nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if linkService == nil {
		// Internal failure --> 504
		core.LogWarn(f, "Transport error when creating face ", params.URI)
		response = mgmt.MakeControlResponse(504, "Transport error when creating face", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	core.LogInfo(f, "Created face with URI ", params.URI)
	responseParams := mgmt.MakeControlParameters()
	f.fillFaceProperties(responseParams, linkService)
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: ", err)
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) update(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in ", interest.Name())
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

	faceID := inFace
	if params.FaceID != nil && *params.FaceID != 0 {
		faceID = *params.FaceID
	}

	// Validate parameters

	responseParams := mgmt.MakeControlParameters()
	areParamsValid := true

	selectedFace := face.FaceTable.Get(faceID)
	if selectedFace == nil {
		core.LogWarn(f, "Cannot update specified (or implicit) FaceID=", faceID, " because it does not exist")
		responseParams.FaceID = new(uint64)
		*responseParams.FaceID = faceID
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode response parameters: ", err)
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(404, "Face does not exist", responseParamsWire)
		}
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Can't update null (or internal) faces via management
	if selectedFace.RemoteURI().Scheme() == "null" || selectedFace.RemoteURI().Scheme() == "internal" {
		responseParams.FaceID = new(uint64)
		*responseParams.FaceID = faceID
		responseParamsWire, err := responseParams.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode response parameters: ", err)
			response = mgmt.MakeControlResponse(500, "Internal error", nil)
		} else {
			response = mgmt.MakeControlResponse(401, "Face cannot be updated via management", responseParamsWire)
		}
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.FacePersistency != nil {
		if selectedFace.RemoteURI().Scheme() == "ether" && *params.FacePersistency != uint64(face.PersistencyPermanent) {
			responseParams.FacePersistency = new(uint64)
			*responseParams.FacePersistency = *params.FacePersistency
			areParamsValid = false
		} else if (selectedFace.RemoteURI().Scheme() == "udp4" || selectedFace.RemoteURI().Scheme() == "udp6") && *params.FacePersistency != uint64(face.PersistencyPersistent) && *params.FacePersistency != uint64(face.PersistencyPermanent) {
			responseParams.FacePersistency = new(uint64)
			*responseParams.FacePersistency = *params.FacePersistency
			areParamsValid = false
		} else if selectedFace.LocalURI().Scheme() == "unix" && *params.FacePersistency != uint64(face.PersistencyPersistent) {
			responseParams.FacePersistency = new(uint64)
			*responseParams.FacePersistency = *params.FacePersistency
			areParamsValid = false
		}
	}

	if (params.Flags != nil && params.Mask == nil) || (params.Flags == nil && params.Mask != nil) {
		core.LogWarn(f, "Flags and Mask fields must either both be present or both be not present")
		if params.Flags != nil {
			responseParams.Flags = new(uint64)
			*responseParams.Flags = *params.Flags
		}
		if params.Mask != nil {
			responseParams.Mask = new(uint64)
			*responseParams.Mask = *params.Mask
		}
		areParamsValid = false
	}

	if !areParamsValid {
		response = mgmt.MakeControlResponse(409, "ControlParameters are incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Actually perform face updates
	// Persistency
	if params.FacePersistency != nil {
		// Correctness of FacePersistency already validated
		selectedFace.SetPersistency(face.Persistency(*params.FacePersistency))
	}

	options := selectedFace.(*face.NDNLPLinkService).Options()

	// Congestion
	if params.BaseCongestionMarkingInterval != nil && time.Duration(*params.BaseCongestionMarkingInterval)*time.Nanosecond != options.BaseCongestionMarkingInterval {
		options.BaseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkingInterval) * time.Nanosecond
		core.LogInfo(f, "FaceID=", faceID, ", BaseCongestionMarkingInterval=", options.BaseCongestionMarkingInterval)
	}

	if params.DefaultCongestionThreshold != nil && *params.DefaultCongestionThreshold != options.DefaultCongestionThresholdBytes {
		options.DefaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		core.LogInfo(f, "FaceID=", faceID, ", DefaultCongestionThreshold=", options.DefaultCongestionThresholdBytes, "B")
	}

	// MTU
	if params.MTU != nil {
		oldMTU := selectedFace.MTU()
		newMTU := int(*params.MTU)
		if *params.MTU > tlv.MaxNDNPacketSize {
			newMTU = tlv.MaxNDNPacketSize
		}
		selectedFace.SetMTU(newMTU)
		core.LogInfo(f, "FaceID=", faceID, ", MTU ", oldMTU, " -> ", newMTU)
	}

	// Flags
	if params.Flags != nil {
		// Presence of mask already validated
		flags := *params.Flags
		mask := *params.Mask

		if mask&0x1 == 1 {
			// Update LocalFieldsEnabled
			if flags&0x1 == 1 {
				core.LogInfo(f, "FaceID=", faceID, ", Enabling local fields")
				options.IsConsumerControlledForwardingEnabled = true
				options.IsIncomingFaceIndicationEnabled = true
				options.IsLocalCachePolicyEnabled = true
			} else {
				core.LogInfo(f, "FaceID=", faceID, ", Disabling local fields")
				options.IsConsumerControlledForwardingEnabled = false
				options.IsIncomingFaceIndicationEnabled = false
				options.IsLocalCachePolicyEnabled = false
			}
		}

		if mask>>1&0x01 == 1 {
			// Update LpReliabilityEnabled
			options.IsReliabilityEnabled = flags>>1&0x01 == 1
			if flags>>1&0x01 == 1 {
				core.LogInfo(f, "FaceID=", faceID, ", Enabling LpReliability")
			} else {
				core.LogInfo(f, "FaceID=", faceID, ", Disabling LpReliability")
			}
		}

		if mask>>2&0x01 == 1 {
			// Update CongestionMarkingEnabled
			options.IsCongestionMarkingEnabled = flags>>2&0x01 == 1
			if flags>>2&0x01 == 1 {
				core.LogInfo(f, "FaceID=", faceID, ", Enabling congestion marking")
			} else {
				core.LogInfo(f, "FaceID=", faceID, ", Disabling congestion marking")
			}
		}
	}

	selectedFace.(*face.NDNLPLinkService).SetOptions(options)

	f.fillFaceProperties(responseParams, selectedFace)
	responseParams.URI = nil
	responseParams.LocalURI = nil
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: ", err)
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) destroy(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in ", interest.Name())
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
		core.LogWarn(f, "Missing FaceId in ControlParameters for ", interest.Name())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if face.FaceTable.Get(*params.FaceID) != nil {
		face.FaceTable.Remove(*params.FaceID)
		core.LogInfo(f, "Destroyed face with FaceID=", *params.FaceID)
	} else {
		core.LogInfo(f, "Ignoring attempt to delete non-existent face with FaceID=", *params.FaceID)
	}

	responseParamsWire, err := params.Encode()
	if err != nil {
		core.LogError(f, "Unable to encode response parameters: ", err)
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

	name, _ := ndn.NameFromString(f.manager.localPrefix.String() + "/faces/list")
	segments := mgmt.MakeStatusDataset(name, f.nextFaceDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode face status dataset: ", err)
			return
		}
		f.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(f, "Published face dataset version=", f.nextFaceDatasetVersion, ", containing ", len(segments), " segments")
	f.nextFaceDatasetVersion++
}

func (f *FaceModule) query(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() < f.manager.prefixLength()+3 {
		// Name not long enough to contain FaceQueryFilter
		core.LogWarn(f, "Missing FaceQueryFilter in ", interest.Name())
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

		if filter.FacePersistency != nil && *filter.FacePersistency != uint64(face.Persistency()) {
			continue
		}

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
			core.LogError(f, "Unable to encode face query dataset: ", err)
			return
		}
		f.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(f, "Published face query dataset version=", f.nextFaceDatasetVersion, ", containing ", len(segments), " segments")
	f.nextFaceDatasetVersion++
}

func (f *FaceModule) createDataset(selectedFace face.LinkService) []byte {
	faceDataset := mgmt.MakeFaceStatus()
	faceDataset.FaceID = uint64(selectedFace.FaceID())
	faceDataset.URI = selectedFace.RemoteURI()
	faceDataset.LocalURI = selectedFace.LocalURI()
	if selectedFace.ExpirationPeriod() != 0 {
		faceDataset.ExpirationPeriod = new(uint64)
		*faceDataset.ExpirationPeriod = uint64(selectedFace.ExpirationPeriod().Milliseconds())
	}
	faceDataset.FaceScope = uint64(selectedFace.Scope())
	faceDataset.FacePersistency = uint64(selectedFace.Persistency())
	faceDataset.LinkType = uint64(selectedFace.LinkType())
	faceDataset.MTU = new(uint64)
	*faceDataset.MTU = uint64(selectedFace.MTU())
	faceDataset.NInInterests = selectedFace.NInInterests()
	faceDataset.NInData = selectedFace.NInData()
	faceDataset.NInNacks = 0
	faceDataset.NOutInterests = selectedFace.NOutInterests()
	faceDataset.NOutData = selectedFace.NOutData()
	faceDataset.NOutNacks = 0
	faceDataset.NInBytes = selectedFace.NInBytes()
	faceDataset.NOutBytes = selectedFace.NOutBytes()
	linkService, ok := selectedFace.(*face.NDNLPLinkService)
	if ok {
		options := linkService.Options()

		faceDataset.BaseCongestionMarkingInterval = new(uint64)
		*faceDataset.BaseCongestionMarkingInterval = uint64(options.BaseCongestionMarkingInterval.Nanoseconds())
		faceDataset.DefaultCongestionThreshold = new(uint64)
		*faceDataset.DefaultCongestionThreshold = options.DefaultCongestionThresholdBytes

		if options.IsConsumerControlledForwardingEnabled {
			// This one will only be enabled if the other two local fields are enabled (and vice versa)
			faceDataset.Flags += 1 << 0
		}
		if options.IsReliabilityEnabled {
			faceDataset.Flags += 1 << 1
		}
		if options.IsCongestionMarkingEnabled {
			faceDataset.Flags += 1 << 2
		}
	}

	faceDatasetEncoded, err := faceDataset.Encode()
	if err != nil {
		core.LogError(f, "Cannot encode FaceStatus for FaceID=", selectedFace.FaceID(), ": ", err)
		return []byte{}
	}
	faceDatasetWire, err := faceDatasetEncoded.Wire()
	if err != nil {
		core.LogError(f, "Cannot encode FaceStatus for FaceID=", selectedFace.FaceID(), ": ", err)
		return []byte{}
	}
	return faceDatasetWire
}

func (f *FaceModule) channels(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() < f.manager.prefixLength()+2 {
		core.LogWarn(f, "Channel dataset Interest too short: ", interest.Name())
		return
	}

	dataset := make([]byte, 0)
	// UDP channel
	ifaces, err := net.Interfaces()
	if err != nil {
		core.LogWarn(f, "Unable to access channel dataset: ", err)
		return
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			core.LogWarn(f, "Unable to access IP addresses for ", iface.Name, ": ", err)
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
				uri := ndn.MakeUDPFaceURI(ipVersion, path, face.UDPUnicastPort)
				channel := mgmt.MakeChannelStatus(uri)
				channelEncoded, err := channel.Encode()
				if err != nil {
					core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
					continue
				}
				channelWire, err := channelEncoded.Wire()
				if err != nil {
					core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
					continue
				}
				dataset = append(dataset, channelWire...)
			}
		}
	}

	// Unix channel
	uri := ndn.MakeUnixFaceURI(face.UnixSocketPath)
	channel := mgmt.MakeChannelStatus(uri)
	channelEncoded, err := channel.Encode()
	if err != nil {
		core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
		return
	}
	channelWire, err := channelEncoded.Wire()
	if err != nil {
		core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
		return
	}
	dataset = append(dataset, channelWire...)

	segments := mgmt.MakeStatusDataset(interest.Name(), f.nextChannelDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode channel dataset: ", err)
			return
		}
		f.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(f, "Published channel dataset version=", f.nextChannelDatasetVersion, ", containing ", len(segments), " segments")
	f.nextChannelDatasetVersion++
}

func (f *FaceModule) fillFaceProperties(params *mgmt.ControlParameters, selectedFace face.LinkService) {
	params.FaceID = new(uint64)
	*params.FaceID = uint64(selectedFace.FaceID())
	params.URI = selectedFace.RemoteURI()
	params.LocalURI = selectedFace.LocalURI()
	params.FacePersistency = new(uint64)
	*params.FacePersistency = uint64(selectedFace.Persistency())
	params.MTU = new(uint64)
	*params.MTU = uint64(selectedFace.MTU())

	params.Flags = new(uint64)
	*params.Flags = 0
	linkService, ok := selectedFace.(*face.NDNLPLinkService)
	if ok {
		options := linkService.Options()

		params.BaseCongestionMarkingInterval = new(uint64)
		*params.BaseCongestionMarkingInterval = uint64(options.BaseCongestionMarkingInterval.Nanoseconds())
		params.DefaultCongestionThreshold = new(uint64)
		*params.DefaultCongestionThreshold = options.DefaultCongestionThresholdBytes

		if options.IsConsumerControlledForwardingEnabled {
			// This one will only be enabled if the other two local fields are enabled (and vice versa)
			*params.Flags += 1 << 0
		}
		if options.IsReliabilityEnabled {
			*params.Flags += 1 << 1
		}
		if options.IsCongestionMarkingEnabled {
			*params.Flags += 1 << 2
		}
	}
}
