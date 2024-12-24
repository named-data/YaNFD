package table

import enc "github.com/named-data/ndnd/std/encoding"

// Readvertising instances
var readvertisers = make([]RibReadvertise, 0)

type RibReadvertise interface {
	// Advertise a route in the RIB
	Announce(name enc.Name, route *Route)
	// Remove a route from the RIB
	Withdraw(name enc.Name, route *Route)
}

func AddReadvertiser(r RibReadvertise) {
	readvertisers = append(readvertisers, r)
}

func readvertiseAnnounce(name enc.Name, route *Route) {
	for _, r := range readvertisers {
		r.Announce(name, route)
	}
}

func readvertiseWithdraw(name enc.Name, route *Route) {
	for _, r := range readvertisers {
		r.Withdraw(name, route)
	}
}
