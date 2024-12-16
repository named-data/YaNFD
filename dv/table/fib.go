package table

import (
	"github.com/pulsejet/go-ndn-dv/config"
	"github.com/pulsejet/go-ndn-dv/nfdc"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type FibEntry struct {
	// next hop face Id
	FaceId uint64
	// cost in this entry
	Cost uint64
	// previous cost
	prevCost uint64
}

// Get the FIB entry for a name prefix.
// router should be hash of the router name.
func (rib *Rib) GetFibEntries(nt *NeighborTable, router uint64) []FibEntry {
	ribEntry := rib.entries[router]

	var face1 uint64 = 0
	var face2 uint64 = 0
	if ns := nt.GetH(ribEntry.nextHop1); ns != nil {
		face1 = ns.faceId
	}
	if ns := nt.GetH(ribEntry.nextHop2); ns != nil {
		face2 = ns.faceId
	}

	return []FibEntry{{
		FaceId: face1,
		Cost:   ribEntry.lowest1,
	}, {
		FaceId: face2,
		Cost:   ribEntry.lowest2,
	}}
}

type Fib struct {
	config   *config.Config
	nfdc     *nfdc.NfdMgmtThread
	names    map[uint64]enc.Name
	prefixes map[uint64][]FibEntry
	mark     map[uint64]bool
}

func NewFib(config *config.Config, nfdc *nfdc.NfdMgmtThread) *Fib {
	return &Fib{
		config:   config,
		nfdc:     nfdc,
		names:    make(map[uint64]enc.Name),
		prefixes: make(map[uint64][]FibEntry),
		mark:     make(map[uint64]bool),
	}
}

func (fib *Fib) Update(name enc.Name, newEntries []FibEntry) bool {
	return fib.UpdateH(name.Hash(), name, newEntries)
}

func (fib *Fib) UpdateH(nameH uint64, name enc.Name, newEntries []FibEntry) bool {
	if _, ok := fib.names[nameH]; !ok {
		fib.names[nameH] = name
	}

	// Set cost of all current entries to infinite and store existing params
	oldEntries := fib.prefixes[nameH]
	for oi := range oldEntries {
		oldEntries[oi].prevCost = oldEntries[oi].Cost
		oldEntries[oi].Cost = config.CostInfinity
	}

	// Merge new entries into old entries
	for _, newEntry := range newEntries {
		// Likely from RIB computation
		if newEntry.Cost >= config.CostInfinity {
			continue
		}

		// Check if same face already exists, in that case
		// just update the cost and other parameters
		found := false
		for oi := range oldEntries {
			if oldEntries[oi].FaceId == newEntry.FaceId {
				// Use the lowest cost known to us. This is needed since the
				// new entries may have some duplicates, e.g. multi-homed prefixes
				oldEntries[oi].Cost = min(newEntry.Cost, oldEntries[oi].Cost)

				// oldEntries is guaranteed to not have duplicates
				found = true
				break
			}
		}

		// If a matching face entry was not found, add the new one
		if !found {
			newEntry.prevCost = config.CostInfinity
			oldEntries = append(oldEntries, newEntry)
		}
	}

	// Unregister entries that are not reachable
	finalEntries := make([]FibEntry, 0, len(oldEntries))
	for _, oldEntry := range oldEntries {
		if oldEntry.Cost >= config.CostInfinity {
			fib.nfdc.Exec(nfdc.NfdMgmtCmd{
				Module: "rib",
				Cmd:    "unregister",
				Args: &mgmt.ControlArgs{
					Name:   name,
					FaceId: utils.IdPtr(oldEntry.FaceId),
					Origin: utils.IdPtr(config.NlsrOrigin),
				},
				Retries: 3,
			})
		} else {
			finalEntries = append(finalEntries, oldEntry)
		}
	}

	// Update all current entries
	for _, entry := range finalEntries {
		// If all params are the same, skip
		if entry.Cost == entry.prevCost {
			continue
		}

		fib.nfdc.Exec(nfdc.NfdMgmtCmd{
			Module: "rib",
			Cmd:    "register",
			Args: &mgmt.ControlArgs{
				Name:   name,
				FaceId: utils.IdPtr(entry.FaceId),
				Cost:   utils.IdPtr(entry.Cost),
				Origin: utils.IdPtr(config.NlsrOrigin),
			},
			Retries: 3,
		})
	}

	if len(finalEntries) > 0 {
		fib.prefixes[nameH] = finalEntries
		return true
	} else {
		delete(fib.prefixes, nameH)
		delete(fib.mark, nameH)
		delete(fib.names, nameH)
		return false
	}
}

func (fib *Fib) MarkH(name uint64) {
	fib.mark[name] = true
}

func (fib *Fib) UnmarkAll() {
	for hash := range fib.mark {
		delete(fib.mark, hash)
	}
}

func (fib *Fib) RemoveUnmarked() {
	for nh := range fib.prefixes {
		if !fib.mark[nh] {
			if name := fib.names[nh]; name != nil {
				fib.UpdateH(nh, name, nil)
			}
		}
	}
}
