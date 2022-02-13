package table

import (
	"testing"

	"github.com/named-data/YaNFD/ndn"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	name, _ := ndn.NameFromString("/something")
	bpe := basePitEntry{
		name: name,
	}
	assert.Equal(t, bpe.Name(), name)
}

func TestCanBePrefix(t *testing.T) {
	bpe := basePitEntry{
		canBePrefix: true,
	}
	assert.Equal(t, bpe.CanBePrefix(), true)
}
