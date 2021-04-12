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
	"github.com/eric135/YaNFD/dispatch"
	"github.com/eric135/YaNFD/fw"
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/mgmt"
	"github.com/eric135/YaNFD/table"
)

// ForwarderStatusModule is the module that provide forwarder status information.
type ForwarderStatusModule struct {
	manager                   *Thread
	nextGeneralDatasetVersion uint64
}

func (f *ForwarderStatusModule) String() string {
	return "ForwarderStatusMgmt"
}

func (f *ForwarderStatusModule) registerManager(manager *Thread) {
	f.manager = manager
}

func (f *ForwarderStatusModule) getManager() *Thread {
	return f.manager
}

func (f *ForwarderStatusModule) handleIncomingInterest(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	// Only allow from /localhost
	if !f.manager.localPrefix.PrefixOf(interest.Name()) {
		core.LogWarn(f, "Received forwarder status management Interest from non-local source - DROP")
		return
	}

	// Dispatch by verb
	verb := interest.Name().At(f.manager.prefixLength() + 1).String()
	switch verb {
	case "general":
		f.general(interest, pitToken, inFace)
	default:
		core.LogWarn(f, "Received Interest for non-existent verb '"+verb+"'")
		response := mgmt.MakeControlResponse(501, "Unknown verb", nil)
		f.manager.sendResponse(response, interest, pitToken, inFace)
		return
	}
}

func (f *ForwarderStatusModule) general(interest *ndn.Interest, pitToken []byte, inFace uint64) {
	if interest.Name().Size() > f.manager.prefixLength()+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	status := mgmt.MakeGeneralStatus()
	status.NfdVersion = core.Version
	status.StartTimestamp = uint64(core.StartTimestamp.UnixNano() / 1000 / 1000)
	status.CurrentTimestamp = uint64(time.Now().UnixNano() / 1000 / 1000)
	// Don't set NNameTreeEntries because we don't use a NameTree
	status.NFibEntries = uint64(len(table.FibStrategyTable.GetAllFIBEntries()))
	for threadID := 0; threadID < fw.NumFwThreads; threadID++ {
		thread := dispatch.GetFWThread(threadID)
		status.NPitEntries += uint64(thread.GetNumPitEntries())
		status.NCsEntries += uint64(thread.GetNumCsEntries())
		status.NInInterests += thread.(*fw.Thread).NInInterests
		status.NInData += thread.(*fw.Thread).NInData
		status.NOutInterests += thread.(*fw.Thread).NOutInterests
		status.NOutData += thread.(*fw.Thread).NOutData
		status.NSatisfiedInterests += thread.(*fw.Thread).NSatisfiedInterests
		status.NUnsatisfiedInterests += thread.(*fw.Thread).NUnsatisfiedInterests
	}

	wire, err := status.Encode()
	if err != nil {
		core.LogError(f, "Cannot encode forwarder status dataset: "+err.Error())
		return
	}
	dataset := wire.Value()

	name, _ := ndn.NameFromString(f.manager.localPrefix.String() + "/status/general")
	segments := mgmt.MakeStatusDataset(name, f.nextGeneralDatasetVersion, dataset)
	for _, segment := range segments {
		encoded, err := segment.Encode()
		if err != nil {
			core.LogError(f, "Unable to encode forwarder status dataset: "+err.Error())
			return
		}
		f.manager.transport.Send(encoded, pitToken, nil)
	}

	core.LogTrace(f, "Published forwarder status dataset version="+strconv.FormatUint(f.nextGeneralDatasetVersion, 10)+", containing "+strconv.Itoa(len(segments))+" segments")
	f.nextGeneralDatasetVersion++
}
