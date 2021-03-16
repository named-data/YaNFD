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
	"github.com/eric135/YaNFD/fw"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/table"
)

// StrategyChoiceModule is the module that handles Strategy Choice Management.
type StrategyChoiceModule struct {
	manager                    *Thread
	nextStrategyDatasetVersion uint64
	strategyPrefix             *ndn.Name
}

func (s *StrategyChoiceModule) String() string {
	return "StrategyChoiceMgmt"
}

func (s *StrategyChoiceModule) registerManager(manager *Thread) {
	s.manager = manager
	s.strategyPrefix = s.manager.localPrefix.DeepCopy().Append(ndn.NewGenericNameComponent([]byte("strategy")))
}

func (s *StrategyChoiceModule) getManager() *Thread {
	return s.manager
}

func (s *StrategyChoiceModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	// Only allow from /localhost
	if !s.manager.localPrefix.PrefixOf(interest.Name()) {
		core.LogWarn(s, "Received strategy management Interest from non-local source - DROP")
		return
	}

	// Dispatch by verb
	verb := interest.Name().At(s.manager.prefixLength() + 1).String()
	switch verb {
	case "set":
		s.set(interest, pitToken, inFace)
	case "unset":
		s.unset(interest, pitToken, inFace)
	case "list":
		s.list(interest, pitToken, inFace)
	default:
		core.LogWarn(s, "Received Interest for non-existent verb '"+verb+"'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (s *StrategyChoiceModule) set(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < s.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(s, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(s, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name == nil {
		core.LogWarn(s, "Missing Name in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Strategy == nil {
		core.LogWarn(s, "Missing Strategy in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if !s.strategyPrefix.PrefixOf(params.Strategy) {
		core.LogWarn(s, "Unknown Strategy="+params.Strategy.String()+" in ControlParameters for Interest="+interest.Name().String())
		response = mgmt.MakeControlResponse(404, "Unknown strategy", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	strategyName := params.Strategy.At(s.strategyPrefix.Size()).String()
	availableVersions, ok := fw.StrategyVersions[strategyName]
	if !ok {
		core.LogWarn(s, "Unknown Strategy="+params.Strategy.String()+" in ControlParameters for Interest="+interest.Name().String())
		response = mgmt.MakeControlResponse(404, "Unknown strategy", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	// Add/verify version information for strategy
	strategyVersion := availableVersions[0]
	for _, version := range availableVersions {
		if version > strategyVersion {
			strategyVersion = version
		}
	}
	if params.Strategy.Size() > s.strategyPrefix.Size()+1 && params.Strategy.At(s.strategyPrefix.Size()+1).Type() != tlv.VersionNameComponent {
		core.LogWarn(s, "Unknown Version="+params.Strategy.At(s.strategyPrefix.Size()+1).String()+" for Strategy="+params.Strategy.String()+" in ControlParameters for Interest="+interest.Name().String())
		response = mgmt.MakeControlResponse(404, "Unknown strategy version", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	} else if params.Strategy.Size() > s.strategyPrefix.Size()+1 {
		strategyVersion := params.Strategy.At(s.strategyPrefix.Size() + 1).(*ndn.VersionNameComponent).Version()
		foundMatchingVersion := false
		for _, version := range availableVersions {
			if version == strategyVersion {
				foundMatchingVersion = true
			}
		}
		if !foundMatchingVersion {
			core.LogWarn(s, "Unknown Version="+strconv.FormatUint(strategyVersion, 10)+" for Strategy="+params.Strategy.String()+" in ControlParameters for Interest="+interest.Name().String())
			response = mgmt.MakeControlResponse(404, "Unknown strategy version", nil)
			s.manager.sendResponse(response, interest, pitToken, inFace)
			return
		}
	} else {
		// Add missing version information to strategy name
		params.Strategy.Append(ndn.NewVersionNameComponent(strategyVersion))
	}

	table.FibStrategyTable.SetStrategy(params.Name, params.Strategy)

	core.LogInfo(s, "Set strategy for Name="+params.Name.String()+" to Strategy="+params.Strategy.String())
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParams.Strategy = params.Strategy
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(s, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	s.manager.sendResponse(response, interest, pitToken, inFace)
}

func (s *StrategyChoiceModule) unset(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	var response *mgmt.ControlResponse

	if interest.Name().Size() < s.manager.prefixLength()+3 {
		// Name not long enough to contain ControlParameters
		core.LogWarn(s, "Missing ControlParameters in "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	params := decodeControlParameters(s, interest)
	if params == nil {
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name == nil {
		core.LogWarn(s, "Missing Name in ControlParameters for "+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	if params.Name.Size() == 0 {
		core.LogWarn(s, "Cannot unset strategy for Name=/"+interest.Name().String())
		response = mgmt.MakeControlResponse(400, "ControlParameters is incorrect", nil)
		s.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}

	table.FibStrategyTable.UnsetStrategy(params.Name)

	core.LogInfo(s, "Unset Strategy for Name="+params.Name.String())
	responseParams := mgmt.MakeControlParameters()
	responseParams.Name = params.Name
	responseParamsWire, err := responseParams.Encode()
	if err != nil {
		core.LogError(s, "Unable to encode response parameters: "+err.Error())
		response = mgmt.MakeControlResponse(500, "Internal error", nil)
	} else {
		response = mgmt.MakeControlResponse(200, "OK", responseParamsWire)
	}
	s.manager.sendResponse(response, interest, pitToken, inFace)
}

func (s *StrategyChoiceModule) list(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() > s.manager.prefixLength()+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	// TODO: For thread safety, we should lock the Strategy table from writes until we are done
	entries := table.FibStrategyTable.GetAllStrategyChoices()
	dataset := make([]byte, 0)
	strategyChoiceList := mgmt.MakeStrategyChoiceList()
	for _, fsEntry := range entries {
		strategyChoiceList = append(strategyChoiceList, mgmt.MakeStrategyChoice(fsEntry.Name, fsEntry.GetStrategy()))
	}

	wires, err := strategyChoiceList.Encode()
	if err != nil {
		core.LogError(s, "Cannot encode list of strategy choices: "+err.Error())
		return
	}
	for _, strategyChoice := range wires {
		encoded, err := strategyChoice.Wire()
		if err != nil {
			core.LogError(s, "Cannot encode strategy choice entry: "+err.Error())
			continue
		}
		dataset = append(dataset, encoded...)
	}

	name, _ := ndn.NameFromString(s.manager.localPrefix.String() + "/strategy-choice/list")
	segments := mgmt.MakeStatusDataset(name, s.nextStrategyDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(s, "Unable to encode strategy choice dataset: "+err.Error())
			return
		}
		s.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(s, "Published strategy choice dataset version="+strconv.FormatUint(s.nextStrategyDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	s.nextStrategyDatasetVersion++
}
