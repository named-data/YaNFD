package security

import (
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/utils"
)

// sha256Signer is a Data signer that uses DigestSha256.
type sha256Signer struct{}

func (sha256Signer) SigInfo() (*ndn.SigConfig, error) {
	return &ndn.SigConfig{
		Type:    ndn.SignatureDigestSha256,
		KeyName: nil,
	}, nil
}

func (sha256Signer) EstimateSize() uint {
	return 32
}

func (sha256Signer) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return h.Sum(nil), nil
}

// NewSha256Signer creates a Data signer that uses DigestSha256.
func NewSha256Signer() ndn.Signer {
	return sha256Signer{}
}

// sha256Signer is an Interest signer that uses DigestSha256.
type sha256IntSigner struct {
	timer ndn.Timer
	seq   uint64
}

func (s *sha256IntSigner) SigInfo() (*ndn.SigConfig, error) {
	s.seq++
	return &ndn.SigConfig{
		Type:    ndn.SignatureDigestSha256,
		KeyName: nil,
		Nonce:   s.timer.Nonce(),
		SigTime: utils.IdPtr(s.timer.Now()),
		SeqNum:  utils.IdPtr(s.seq),
	}, nil
}

func (*sha256IntSigner) EstimateSize() uint {
	return 32
}

func (*sha256IntSigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	return sha256Signer{}.ComputeSigValue(covered)
}

// NewSha256IntSigner creates an Interest signer that uses DigestSha256.
func NewSha256IntSigner(timer ndn.Timer) ndn.Signer {
	return &sha256IntSigner{
		timer: timer,
		seq:   0,
	}
}
