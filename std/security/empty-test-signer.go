package security

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// emptySigner is a signer used for test only. It gives an empty signature value.
type emptySigner struct{}

func (emptySigner) SigInfo() (*ndn.SigConfig, error) {
	return &ndn.SigConfig{
		Type:    ndn.SignatureEmptyTest,
		KeyName: nil,
	}, nil
}

func (emptySigner) EstimateSize() uint {
	return 0
}

func (emptySigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	return []byte{}, nil
}

// NewEmptySigner creates an empty signer for test.
func NewEmptySigner() ndn.Signer {
	return emptySigner{}
}
