/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"math"
	"time"

	"github.com/eric135/YaNFD/ndn"
)

// MakeStatusDataset creates a set of status dataset packets based upon the specified prefix, version, and dataset information.
func MakeStatusDataset(name *ndn.Name, version uint64, dataset []byte) []*ndn.Data {
	// Split into 8000 byte segments and publish
	nSegments := int(math.Ceil(float64(len(dataset)) / float64(8000)))
	if nSegments == 0 {
		// Empty dataset
		segmentName := name.DeepCopy().Append(ndn.NewVersionNameComponent(version)).Append(ndn.NewSegmentNameComponent(0))
		data := ndn.NewData(segmentName, []byte{})
		metaInfo := ndn.NewMetaInfo()
		metaInfo.SetFreshnessPeriod(1000 * time.Millisecond)
		metaInfo.SetFinalBlockID(ndn.NewSegmentNameComponent(0))
		data.SetMetaInfo(metaInfo)
		return []*ndn.Data{data}
	}
	segments := make([]*ndn.Data, nSegments)
	for segment := 0; segment < nSegments; segment++ {
		var content []byte
		if segment < segment-1 {
			content = dataset[8000*segment : 8000*(segment+1)]
		} else {
			content = dataset[8000*segment:]
		}
		segmentName := name.DeepCopy().Append(ndn.NewVersionNameComponent(version)).Append(ndn.NewSegmentNameComponent(uint64(segment)))
		data := ndn.NewData(segmentName, content)
		metaInfo := ndn.NewMetaInfo()
		metaInfo.SetFreshnessPeriod(1000 * time.Millisecond)
		metaInfo.SetFinalBlockID(ndn.NewSegmentNameComponent(uint64(nSegments - 1)))
		data.SetMetaInfo(metaInfo)
		segments[segment] = data
	}
	return segments
}
