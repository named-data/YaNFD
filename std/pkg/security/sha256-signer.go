package security

import (
	"crypto/sha256"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

type sha256Signer struct{}

func (_ sha256Signer) SigInfo(ndn.Data) (*ndn.SigConfig, error) {
	return &ndn.SigConfig{
		Type:    ndn.SignatureDigestSha256,
		KeyName: nil,
	}, nil
}

func (_ sha256Signer) EstimateSize() uint {
	return 32
}

func (_ sha256Signer) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return h.Sum(nil), nil
}

func NewSha256Signer() ndn.Signer {
	return sha256Signer{}
}
