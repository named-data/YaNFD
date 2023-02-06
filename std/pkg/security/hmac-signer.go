package security

import (
	"crypto/hmac"
	"crypto/sha256"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// hmacSigner is a Data signer that uses a provided HMAC key.
type hmacSigner struct {
	key []byte
}

func (hmacSigner) SigInfo() (*ndn.SigConfig, error) {
	return &ndn.SigConfig{
		Type:    ndn.SignatureHmacWithSha256,
		KeyName: nil,
	}, nil
}

func (hmacSigner) EstimateSize() uint {
	return 32
}

func (signer hmacSigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	mac := hmac.New(sha256.New, signer.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return mac.Sum(nil), nil
}

// NewHmacSigner creates a Data signer that uses DigestSha256.
func NewHmacSigner(key []byte) ndn.Signer {
	return hmacSigner{key}
}

// hmacIntSigner is a Interest signer that uses a provided HMAC key.
type hmacIntSigner struct {
	key   []byte
	timer ndn.Timer
	seq   uint64
}

func (s hmacIntSigner) SigInfo() (*ndn.SigConfig, error) {
	return &ndn.SigConfig{
		Type:    ndn.SignatureHmacWithSha256,
		KeyName: enc.Name{enc.Component{Typ: enc.TypeGenericNameComponent, Val: s.key}},
		Nonce:   s.timer.Nonce(),
		SigTime: utils.IdPtr(s.timer.Now()),
		SeqNum:  utils.IdPtr(s.seq),
	}, nil
}

func (hmacIntSigner) EstimateSize() uint {
	return 32
}

func (signer hmacIntSigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	mac := hmac.New(sha256.New, signer.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return mac.Sum(nil), nil
}

// NewHmacIntSigner creates an Interest signer that uses DigestSha256.
func NewHmacIntSigner(key []byte, timer ndn.Timer) ndn.Signer {
	return hmacIntSigner{key, timer, 0}
}
