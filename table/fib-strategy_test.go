package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

func TestFibStrategyEntryGetters(t *testing.T) {
	name, _ := enc.NameFromStr("/something")

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
		component: name[0],
		name:      name,
		nexthops:  nextHops,
		strategy:  name,
	}

	assert.True(t, bfse.Name().Equal(name))
	assert.True(t, bfse.GetStrategy().Equal(name))
	assert.Equal(t, 2, len(bfse.GetNextHops()))
}
