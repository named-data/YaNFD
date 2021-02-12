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
	segments := make([]*ndn.Data, nSegments)
	for segment := 0; segment < nSegments; segment++ {
		var content []byte
		if segment < segment-1 {
			content = dataset[8000*segment : 8000*(segment+1)]
		} else {
			content = dataset[8000*segment:]
		}
		name.Append(ndn.NewVersionNameComponent(version)).Append(ndn.NewSegmentNameComponent(uint64(segment)))
		data := ndn.NewData(name, content)
		metaInfo := ndn.NewMetaInfo()
		metaInfo.SetFreshnessPeriod(1000 * time.Millisecond)
		metaInfo.SetFinalBlockID(ndn.NewSegmentNameComponent(uint64(nSegments - 1)))
		data.SetMetaInfo(metaInfo)
		segments[segment] = data
	}
	return segments
}
