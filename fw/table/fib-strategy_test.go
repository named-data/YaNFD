package table

import (
	"testing"

	"github.com/named-data/YaNFD/ndn"

	"github.com/stretchr/testify/assert"
)

func TestFibStrategyEntryGetters(t *testing.T) {
	name, _ := ndn.NameFromString("/something")

	nextHop1 := FibNextHopEntry{
		Nexthop: 100,
		Cost:    101,
	}

	nextHop2 := FibNextHopEntry{
		Nexthop: 102,
		Cost:    103,
	}

	nextHops := []*FibNextHopEntry{&nextHop1, &nextHop2}

	bfse := baseFibStrategyEntry{
		component: name.At(0),
		name:      name,
		nexthops:  nextHops,
		strategy:  name,
	}

	assert.True(t, bfse.Name().Equals(name))
	assert.True(t, bfse.GetStrategy().Equals(name))
	assert.Equal(t, 2, len(bfse.GetNextHops()))
}
