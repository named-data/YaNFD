package table

import enc "github.com/zjkmxy/go-ndn/pkg/encoding"

// Readvertising instances
var readvertisers = make([]RibReadvertise, 0)

type RibReadvertise interface {
	// Advertise a route in the RIB
	Announce(name enc.Name, route *Route)
	// Remove a route from the RIB
	Withdraw(name enc.Name, faceID uint64, origin uint64)
}

func AddReadvertiser(r RibReadvertise) {
	readvertisers = append(readvertisers, r)
}

func (r *RibTable) readvertiseAnnounce(name enc.Name, route *Route) {
	for _, r := range readvertisers {
		r.Announce(name, route)
	}
}

func (r *RibTable) readvertiseWithdraw(name enc.Name, faceID uint64, origin uint64) {
	for _, r := range readvertisers {
		r.Withdraw(name, faceID, origin)
	}
}
