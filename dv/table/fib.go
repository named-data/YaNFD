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

// Check if the given list has a matching entry with the same face
func (fe FibEntry) isSameFaceIn(entries []FibEntry) FibEntry {
	for _, entry := range entries {
		if fe.FaceId == entry.FaceId {
			return fe
		}
	}
	return FibEntry{}
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

func (fib *Fib) Update(name enc.Name, entries []FibEntry) bool {
	nameH := name.Hash()
	if _, ok := fib.names[nameH]; !ok {
		fib.names[nameH] = name
	}

	final := make([]FibEntry, 0, len(entries))

	// Unregister old entries
	for _, entry := range fib.prefixes[nameH] {
		// If same faceId is present in any other entry, do not unregister
		prev := entry.isSameFaceIn(entries)
		if prev.FaceId != 0 {
			// If the cost is the same, skip the registration too
			if prev.Cost == entry.Cost {
				entry.FaceId = 0
			}
			final = append(final, prev)
			continue
		}

		fib.nfdc.Exec(nfdc.NfdMgmtCmd{
			Module: "rib",
			Cmd:    "unregister",
			Args: &mgmt.ControlArgs{
				Name:   name,
				FaceId: utils.IdPtr(entry.FaceId),
			},
			Retries: 3,
		})
	}

	// Register new entries
	for _, entry := range entries {
		if entry.FaceId == 0 || entry.Cost >= config.CostInfinity {
			continue
		}

		final = append(final, entry)

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

	if len(final) > 0 {
		fib.prefixes[nameH] = final
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
				fib.Update(name, nil)
			}
		}
	}
}
