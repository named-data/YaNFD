/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2022 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"math"
	"net"
	"sort"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face"
	oldndn "github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/tlv"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
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
	face.FaceEventSendFunc = f.sendFaceEventNotification
}

func (f *FaceModule) getManager() *Thread {
	return f.manager
}

func (f *FaceModule) handleIncomingInterest(interest *spec.Interest, pitToken []byte, inFace uint64) {
	// Only allow from /localhost
	if !f.manager.localPrefix.IsPrefix(interest.NameV) {
		core.LogWarn(f, "Received face management Interest from non-local source - DROP")
		return
	}

	// Dispatch by verb
	verb := interest.NameV[f.manager.prefixLength()+1].String()
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
	// case "channels":
	// 	f.channels(interest, pitToken, inFace)
	case "events":
		f.events(interest, pitToken, inFace)
	default:
		core.LogWarn(f, "Received Interest for non-existent verb '", verb, "'")
		response := makeControlResponse(501, "Unknown verb", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (f *FaceModule) create(interest *spec.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if len(interest.NameV) < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in ", interest.Name())
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(f, interest)
	if params == nil {
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Uri == nil {
		core.LogWarn(f, "Missing URI in ControlParameters for ", interest.Name())
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	URI := oldndn.DecodeURIString(*params.Uri)
	if URI == nil || URI.Canonize() != nil {
		core.LogWarn(f, "Cannot canonize remote URI in ControlParameters for ", interest.Name())
		response = makeControlResponse(406, "URI could not be canonized", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if (params.Flags != nil && params.Mask == nil) || (params.Flags == nil && params.Mask != nil) {
		core.LogWarn(f, "Flags and Mask fields either both be present or both be not present")
		response = makeControlResponse(409, "Incomplete Flags/Mask combination", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Ensure does not conflict with existing face
	existingFace := face.FaceTable.GetByURI(URI)
	if existingFace != nil {
		core.LogWarn(f, "Cannot create face ", URI, ": Conflicts with existing face FaceID=",
			existingFace.FaceID(), ", RemoteURI=", existingFace.RemoteURI())
		responseParams := map[string]any{}
		f.fillFaceProperties(responseParams, existingFace)
		response = makeControlResponse(409, "Conflicts with existing face", responseParams)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	var linkService *face.NDNLPLinkService

	if URI.Scheme() == "udp4" || URI.Scheme() == "udp6" {
		// Check that remote endpoint is not a unicast address
		if remoteAddr := net.ParseIP(URI.Path()); remoteAddr != nil && !remoteAddr.IsGlobalUnicast() && !remoteAddr.IsLinkLocalUnicast() {
			core.LogWarn(f, "Cannot create unicast UDP face to non-unicast address ", URI)
			response = makeControlResponse(406, "URI must be unicast", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		// Validate and populate missing optional params
		// TODO: Validate and use LocalURI if present
		/*if params.LocalURI != nil {
			if params.LocalURI.Canonize() != nil {
				core.LogWarn(f, "Cannot canonize local URI in ControlParameters for ", interest.Name())
				response = makeControlResponse(406, "LocalURI could not be canonized", nil)
				return
			}
			if params.LocalURI.Scheme() != params.URI.Scheme() {
				core.LogWarn(f, "Local URI scheme does not match remote URI scheme in ControlParameters for ", interest.Name())
				response = makeControlResponse(406, "LocalURI scheme does not match URI scheme", nil)
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
			response = makeControlResponse(406, "Unacceptable persistency", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		baseCongestionMarkingInterval := 100 * time.Millisecond
		if params.BaseCongestionMarkInterval != nil {
			baseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkInterval) * time.Nanosecond
		}

		defaultCongestionThresholdBytes := uint64(math.Pow(2, 16))
		if params.DefaultCongestionThreshold != nil {
			defaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		}

		// Create new UDP face
		transport, err := face.MakeUnicastUDPTransport(URI, nil, persistency)
		if err != nil {
			core.LogWarn(f, "Unable to create unicast UDP face with URI ", URI, ":", err.Error())
			response = makeControlResponse(406, "Transport error", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		if params.Mtu != nil {
			mtu := int(*params.Mtu)
			if *params.Mtu > tlv.MaxNDNPacketSize {
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

			if mask&face.FaceFlagLocalFields > 0 {
				// LocalFieldsEnabled
				if flags&face.FaceFlagLocalFields > 0 {
					options.IsConsumerControlledForwardingEnabled = true
					options.IsIncomingFaceIndicationEnabled = true
					options.IsLocalCachePolicyEnabled = true
				} else {
					options.IsConsumerControlledForwardingEnabled = false
					options.IsIncomingFaceIndicationEnabled = false
					options.IsLocalCachePolicyEnabled = false
				}
			}

			if mask&face.FaceFlagLpReliabilityEnabled > 0 {
				// LpReliabilityEnabled
				options.IsReliabilityEnabled = flags&face.FaceFlagLpReliabilityEnabled > 0
			}

			// Congestion control
			if mask&face.FaceFlagCongestionMarking > 0 {
				// CongestionMarkingEnabled
				options.IsCongestionMarkingEnabled = flags&face.FaceFlagCongestionMarking > 0
			}
			options.BaseCongestionMarkingInterval = baseCongestionMarkingInterval
			options.DefaultCongestionThresholdBytes = defaultCongestionThresholdBytes
		}

		linkService = face.MakeNDNLPLinkService(transport, options)
		face.FaceTable.Add(linkService)

		// Start new face
		go linkService.Run(nil)
	} else if URI.Scheme() == "tcp4" || URI.Scheme() == "tcp6" {
		// Check that remote endpoint is not a unicast address
		if remoteAddr := net.ParseIP(URI.Path()); remoteAddr != nil && !remoteAddr.IsGlobalUnicast() && !remoteAddr.IsLinkLocalUnicast() {
			core.LogWarn(f, "Cannot create unicast TCP face to non-unicast address ", URI)
			response = makeControlResponse(406, "URI must be unicast", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		persistency := face.PersistencyPersistent
		if params.FacePersistency != nil && (*params.FacePersistency == uint64(face.PersistencyPersistent) || *params.FacePersistency == uint64(face.PersistencyPermanent)) {
			persistency = face.Persistency(*params.FacePersistency)
		} else if params.FacePersistency != nil {
			core.LogWarn(f, "Unacceptable persistency ", face.Persistency(*params.FacePersistency), " for UDP face specified in ControlParameters for ", interest.Name())
			response = makeControlResponse(406, "Unacceptable persistency", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		baseCongestionMarkingInterval := 100 * time.Millisecond
		if params.BaseCongestionMarkInterval != nil {
			baseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkInterval) * time.Nanosecond
		}

		defaultCongestionThresholdBytes := uint64(math.Pow(2, 16))
		if params.DefaultCongestionThreshold != nil {
			defaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		}

		// Create new TCP face
		transport, err := face.MakeUnicastTCPTransport(URI, nil, persistency)
		if err != nil {
			core.LogWarn(f, "Unable to create unicast TCP face with URI ", URI, ":", err.Error())
			response = makeControlResponse(406, "Transport error", nil)
			f.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}

		if params.Mtu != nil {
			mtu := int(*params.Mtu)
			if *params.Mtu > tlv.MaxNDNPacketSize {
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

			if mask&face.FaceFlagLocalFields > 0 {
				// LocalFieldsEnabled
				if flags&face.FaceFlagLocalFields > 0 {
					options.IsConsumerControlledForwardingEnabled = true
					options.IsIncomingFaceIndicationEnabled = true
					options.IsLocalCachePolicyEnabled = true
				} else {
					options.IsConsumerControlledForwardingEnabled = false
					options.IsIncomingFaceIndicationEnabled = false
					options.IsLocalCachePolicyEnabled = false
				}
			}

			if mask&face.FaceFlagLpReliabilityEnabled > 0 {
				// LpReliabilityEnabled
				options.IsReliabilityEnabled = flags&face.FaceFlagLpReliabilityEnabled > 0
			}

			// Congestion control
			if mask&face.FaceFlagCongestionMarking > 0 {
				// CongestionMarkingEnabled
				options.IsCongestionMarkingEnabled = flags&face.FaceFlagCongestionMarking > 0
			}
			options.BaseCongestionMarkingInterval = baseCongestionMarkingInterval
			options.DefaultCongestionThresholdBytes = defaultCongestionThresholdBytes
		}

		linkService = face.MakeNDNLPLinkService(transport, options)
		face.FaceTable.Add(linkService)

		// Start new face
		go linkService.Run(nil)
	} else {
		// Unsupported scheme
		core.LogWarn(f, "Cannot create face with URI ", URI, ": Unsupported scheme ", URI)
		response = makeControlResponse(406, "Unsupported scheme "+URI.Scheme(), nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if linkService == nil {
		// Internal failure --> 504
		core.LogWarn(f, "Transport error when creating face ", URI)
		response = makeControlResponse(504, "Transport error when creating face", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	core.LogInfo(f, "Created face with URI ", URI)
	responseParams := map[string]any{}
	f.fillFaceProperties(responseParams, linkService)
	response = makeControlResponse(200, "OK", responseParams)
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) update(interest *spec.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if len(interest.NameV) < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in ", interest.Name())
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(f, interest)
	if params == nil {
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	faceID := inFace
	if params.FaceId != nil && *params.FaceId != 0 {
		faceID = *params.FaceId
	}

	// Validate parameters

	responseParams := map[string]any{}
	areParamsValid := true

	selectedFace := face.FaceTable.Get(faceID)
	if selectedFace == nil {
		core.LogWarn(f, "Cannot update specified (or implicit) FaceID=", faceID, " because it does not exist")
		responseParams["FaceId"] = uint64(faceID)
		response = makeControlResponse(404, "Face does not exist", responseParams)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Can't update null (or internal) faces via management
	if selectedFace.RemoteURI().Scheme() == "null" || selectedFace.RemoteURI().Scheme() == "internal" {
		responseParams["FaceId"] = uint64(faceID)
		response = makeControlResponse(401, "Face cannot be updated via management", responseParams)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.FacePersistency != nil {
		if selectedFace.RemoteURI().Scheme() == "ether" && *params.FacePersistency != uint64(face.PersistencyPermanent) {
			responseParams["FacePersistency"] = uint64(*params.FacePersistency)
			areParamsValid = false
		} else if (selectedFace.RemoteURI().Scheme() == "udp4" || selectedFace.RemoteURI().Scheme() == "udp6") &&
			*params.FacePersistency != uint64(face.PersistencyPersistent) &&
			*params.FacePersistency != uint64(face.PersistencyPermanent) {
			responseParams["FacePersistency"] = uint64(*params.FacePersistency)
			areParamsValid = false
		} else if selectedFace.LocalURI().Scheme() == "unix" && *params.FacePersistency != uint64(face.PersistencyPersistent) {
			responseParams["FacePersistency"] = uint64(*params.FacePersistency)
			areParamsValid = false
		}
	}

	if (params.Flags != nil && params.Mask == nil) || (params.Flags == nil && params.Mask != nil) {
		core.LogWarn(f, "Flags and Mask fields must either both be present or both be not present")
		if params.Flags != nil {
			responseParams["Flags"] = uint64(*params.Flags)
		}
		if params.Mask != nil {
			responseParams["Mask"] = uint64(*params.Mask)
		}
		areParamsValid = false
	}

	if !areParamsValid {
		response = makeControlResponse(409, "ControlParameters are incorrect", nil)
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
	if params.BaseCongestionMarkInterval != nil &&
		time.Duration(*params.BaseCongestionMarkInterval)*time.Nanosecond != options.BaseCongestionMarkingInterval {
		options.BaseCongestionMarkingInterval = time.Duration(*params.BaseCongestionMarkInterval) * time.Nanosecond
		core.LogInfo(f, "FaceID=", faceID, ", BaseCongestionMarkingInterval=", options.BaseCongestionMarkingInterval)
	}

	if params.DefaultCongestionThreshold != nil && *params.DefaultCongestionThreshold != options.DefaultCongestionThresholdBytes {
		options.DefaultCongestionThresholdBytes = *params.DefaultCongestionThreshold
		core.LogInfo(f, "FaceID=", faceID, ", DefaultCongestionThreshold=", options.DefaultCongestionThresholdBytes, "B")
	}

	// MTU
	if params.Mtu != nil {
		oldMTU := selectedFace.MTU()
		newMTU := int(*params.Mtu)
		if *params.Mtu > tlv.MaxNDNPacketSize {
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

		if mask&face.FaceFlagLocalFields > 0 {
			// Update LocalFieldsEnabled
			if flags&face.FaceFlagLocalFields > 0 {
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

		if mask&face.FaceFlagLpReliabilityEnabled > 0 {
			// Update LpReliabilityEnabled
			options.IsReliabilityEnabled = flags&face.FaceFlagLpReliabilityEnabled > 0
			if flags&face.FaceFlagLpReliabilityEnabled > 0 {
				core.LogInfo(f, "FaceID=", faceID, ", Enabling LpReliability")
			} else {
				core.LogInfo(f, "FaceID=", faceID, ", Disabling LpReliability")
			}
		}

		if mask&face.FaceFlagCongestionMarking > 0 {
			// Update CongestionMarkingEnabled
			options.IsCongestionMarkingEnabled = flags&face.FaceFlagCongestionMarking > 0
			if flags&face.FaceFlagCongestionMarking > 0 {
				core.LogInfo(f, "FaceID=", faceID, ", Enabling congestion marking")
			} else {
				core.LogInfo(f, "FaceID=", faceID, ", Disabling congestion marking")
			}
		}
	}

	selectedFace.(*face.NDNLPLinkService).SetOptions(options)

	f.fillFaceProperties(responseParams, selectedFace)
	delete(responseParams, "Uri")
	delete(responseParams, "LocalUri")
	response = makeControlResponse(200, "OK", responseParams)
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) destroy(interest *spec.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if len(interest.NameV) < f.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(f, "Missing ControlParameters in ", interest.Name())
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(f, interest)
	if params == nil {
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.FaceId == nil {
		core.LogWarn(f, "Missing FaceId in ControlParameters for ", interest.Name())
		response = makeControlResponse(400, "ControlParameters is incorrect", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if face.FaceTable.Get(*params.FaceId) != nil {
		face.FaceTable.Remove(*params.FaceId)
		core.LogInfo(f, "Destroyed face with FaceID=", *params.FaceId)
	} else {
		core.LogInfo(f, "Ignoring attempt to delete non-existent face with FaceID=", *params.FaceId)
	}

	response = makeControlResponse(200, "OK", params.ToDict())
	f.manager.sendResponse(response, interest, pitToken, inFace)
}

func (f *FaceModule) list(interest *spec.Interest, pitToken []byte, inFace uint64) {
	if len(interest.NameV) > f.manager.prefixLength()+2 {
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
	dataset := enc.Wire{}
	for _, faceID := range faceIDs {
		dataset = append(dataset, f.createDataset(faces[faceID])...)
	}

	name, _ := enc.NameFromStr(f.manager.localPrefix.String() + "/faces/list")
	segments := makeStatusDataset(name, f.nextFaceDatasetVersion, dataset)
	f.manager.transport.Send(segments, pitToken, nil)

	core.LogTrace(f, "Published face dataset version=", f.nextFaceDatasetVersion,
		", containing ", len(segments), " segments")
	f.nextFaceDatasetVersion++
}

func (f *FaceModule) query(interest *spec.Interest, pitToken []byte, inFace uint64) {
	if len(interest.NameV) < f.manager.prefixLength()+3 {
		// Name not long enough to contain FaceQueryFilter
		core.LogWarn(f, "Missing FaceQueryFilter in ", interest.Name())
		return
	}
	filterV, err := mgmt.ParseFaceQueryFilter(enc.NewBufferReader(interest.NameV[f.manager.prefixLength()+2].Val), true)
	if err != nil {
		return
	}
	filter := filterV.Val

	faces := face.FaceTable.GetAll()
	matchingFaces := make([]int, 0)
	for pos, face := range faces {
		if filter.FaceId != nil && *filter.FaceId != face.FaceID() {
			continue
		}

		if filter.UriScheme != nil &&
			*filter.UriScheme != face.LocalURI().Scheme() &&
			*filter.UriScheme != face.RemoteURI().Scheme() {
			continue
		}

		if filter.Uri != nil && *filter.Uri != face.RemoteURI().String() {
			continue
		}

		if filter.LocalUri != nil && *filter.LocalUri != face.LocalURI().String() {
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

	dataset := enc.Wire{}
	for _, pos := range matchingFaces {
		dataset = append(dataset, f.createDataset(faces[pos])...)
	}

	segments := makeStatusDataset(interest.Name(), f.nextFaceDatasetVersion, dataset)
	f.manager.transport.Send(segments, pitToken, nil)

	core.LogTrace(f, "Published face query dataset version=", f.nextFaceDatasetVersion,
		", containing ", len(segments), " segments")
	f.nextFaceDatasetVersion++
}

func (f *FaceModule) createDataset(selectedFace face.LinkService) enc.Wire {
	faceDataset := &mgmt.FaceStatus{
		FaceId:          selectedFace.FaceID(),
		Uri:             selectedFace.RemoteURI().String(),
		LocalUri:        selectedFace.LocalURI().String(),
		FaceScope:       uint64(selectedFace.Scope()),
		FacePersistency: uint64(selectedFace.Persistency()),
		LinkType:        uint64(selectedFace.LinkType()),
		Mtu:             utils.IdPtr(uint64(selectedFace.MTU())),
		NInInterests:    selectedFace.NInInterests(),
		NInData:         selectedFace.NInData(),
		NInNacks:        0,
		NOutInterests:   selectedFace.NOutInterests(),
		NOutData:        selectedFace.NOutData(),
		NOutNacks:       0,
		NInBytes:        selectedFace.NInBytes(),
		NOutBytes:       selectedFace.NInBytes(),
	}
	if selectedFace.ExpirationPeriod() != 0 {
		faceDataset.ExpirationPeriod = utils.IdPtr(uint64(selectedFace.ExpirationPeriod().Milliseconds()))
	}
	linkService, ok := selectedFace.(*face.NDNLPLinkService)
	if ok {
		options := linkService.Options()

		faceDataset.BaseCongestionMarkInterval = utils.IdPtr(uint64(options.BaseCongestionMarkingInterval.Nanoseconds()))
		faceDataset.DefaultCongestionThreshold = utils.IdPtr(options.DefaultCongestionThresholdBytes)
		faceDataset.Flags = options.Flags()
		if options.IsConsumerControlledForwardingEnabled {
			// This one will only be enabled if the other two local fields are enabled (and vice versa)
			faceDataset.Flags |= face.FaceFlagLocalFields
		}
		if options.IsReliabilityEnabled {
			faceDataset.Flags |= face.FaceFlagLpReliabilityEnabled
		}
		if options.IsCongestionMarkingEnabled {
			faceDataset.Flags |= face.FaceFlagCongestionMarking
		}
	}

	return faceDataset.Encode()
}

// func (f *FaceModule) channels(interest *spec.Interest, pitToken []byte, inFace uint64) {
// 	if len(interest.NameV) < f.manager.prefixLength()+2 {
// 		core.LogWarn(f, "Channel dataset Interest too short: ", interest.Name())
// 		return
// 	}

// 	dataset := make([]byte, 0)
// 	// UDP channel
// 	ifaces, err := net.Interfaces()
// 	if err != nil {
// 		core.LogWarn(f, "Unable to access channel dataset: ", err)
// 		return
// 	}
// 	for _, iface := range ifaces {
// 		addrs, err := iface.Addrs()
// 		if err != nil {
// 			core.LogWarn(f, "Unable to access IP addresses for ", iface.Name, ": ", err)
// 			return
// 		}
// 		for _, addr := range addrs {
// 			ipAddr := addr.(*net.IPNet)

// 			ipVersion := 4
// 			path := ipAddr.IP.String()
// 			if ipAddr.IP.To4() == nil {
// 				ipVersion = 6
// 				path += "%" + iface.Name
// 			}

// 			if !addr.(*net.IPNet).IP.IsLoopback() {
// 				uri := oldndn.MakeUDPFaceURI(ipVersion, path, face.UDPUnicastPort)
// 				channel := mgmt.ChannelStatus{}
// 				channelEncoded, err := channel.Encode()
// 				if err != nil {
// 					core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
// 					continue
// 				}
// 				channelWire, err := channelEncoded.Wire()
// 				if err != nil {
// 					core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
// 					continue
// 				}
// 				dataset = append(dataset, channelWire...)
// 			}
// 		}
// 	}

// 	// Unix channel
// 	uri := oldndn.MakeUnixFaceURI(face.UnixSocketPath)
// 	channel := mgmt.MakeChannelStatus(uri)
// 	channelEncoded, err := channel.Encode()
// 	if err != nil {
// 		core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
// 		return
// 	}
// 	channelWire, err := channelEncoded.Wire()
// 	if err != nil {
// 		core.LogError(f, "Cannot encode ChannelStatus for Channel=", uri, ": ", err)
// 		return
// 	}
// 	dataset = append(dataset, channelWire...)

// 	segments := makeStatusDataset(interest.Name(), f.nextChannelDatasetVersion, dataset)
// 	f.manager.transport.Send(segments, pitToken, nil)

// 	core.LogTrace(f, "Published channel dataset version=", f.nextChannelDatasetVersion, ", containing ", len(segments), " segments")
// 	f.nextChannelDatasetVersion++
// }

func (f *FaceModule) fillFaceProperties(params map[string]any, selectedFace face.LinkService) {
	params["FaceId"] = uint64(selectedFace.FaceID())
	params["Uri"] = selectedFace.RemoteURI().String()
	params["LocalUri"] = selectedFace.LocalURI().String()
	params["FacePersistency"] = uint64(selectedFace.Persistency())
	params["Mtu"] = uint64(selectedFace.MTU())
	params["Flags"] = 0
	if linkService, ok := selectedFace.(*face.NDNLPLinkService); ok {
		options := linkService.Options()
		params["BaseCongestionMarkInterval"] = uint64(options.BaseCongestionMarkingInterval.Nanoseconds())
		params["DefaultCongestionThreshold"] = options.DefaultCongestionThresholdBytes
		params["Flags"] = uint64(options.Flags())
	}
}

func (f *FaceModule) events(interest *spec.Interest, pitToken []byte, inFace uint64) {
	var id uint64 = 0
	var err error

	if len(interest.NameV) < f.manager.prefixLength()+3 {
		// Name is a prefix, take the last one
		id = face.FaceEventLastId()
		if !interest.CanBePrefix() {
			core.LogInfo(f, "FaceEvent Interest with a prefix should set CanBePrefix=true: ", interest.Name())
			return
		}
	} else {
		seg := interest.NameV[f.manager.prefixLength()+2]
		if seg.Typ != enc.TypeSegmentNameComponent {
			core.LogInfo(f, "FaceEvent Interest with an illegible event ID: ", interest.Name())
			return
		}
		id, err = tlv.DecodeNNI(seg.Val)
		if err != nil {
			core.LogInfo(f, "FaceEvent Interest with an illegible event ID: ", interest.Name(), "err: ", err)
			return
		}
	}

	f.sendFaceEventNotification(id, pitToken)
}

func (f *FaceModule) sendFaceEventNotification(id uint64, pitToken []byte) {
	event := face.GetFaceEvent(id)
	if event == nil {
		return
	}

	eventBlock, err := event.Encode()
	if err != nil {
		core.LogError(f, "Cannot encode FaceEventNotification for EventID=", id, ": ", err)
		return
	}

	dataName, err := enc.NameFromStr("/localhost/nfd/faces/events")
	if err != nil {
		core.LogError(f, "Cannot encode FaceEventNotification name.")
		return
	}
	dataName = append(dataName, enc.NewSequenceNumComponent(id))
	dataWire, _, err := spec.Spec{}.MakeData(
		dataName,
		&ndn.DataConfig{
			ContentType: utils.IdPtr(ndn.ContentTypeBlob),
			Freshness:   utils.IdPtr(1 * time.Millisecond),
		},
		eventBlock,
		sec.NewSha256Signer(),
	)
	if err != nil {
		core.LogError(f, "Cannot encode FaceEventNotification data for EventID=", id, ": ", err)
		return
	}
	if f.manager.transport != nil {
		f.manager.transport.Send(dataWire, pitToken, nil)
	}
}
