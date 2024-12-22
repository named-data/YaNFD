package security

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"time"

	enc "github.com/pulsejet/ndnd/std/encoding"
	basic_engine "github.com/pulsejet/ndnd/std/engine/basic"
	"github.com/pulsejet/ndnd/std/ndn"
	"github.com/pulsejet/ndnd/std/utils"
)

// rsaSigner is a signer that uses ECC key to sign packets.
type rsaSigner struct {
	timer ndn.Timer
	seq   uint64

	keyLocatorName enc.Name
	key            *rsa.PrivateKey
	keyLen         uint
	forCert        bool
	forInt         bool
	certExpireTime time.Duration
}

func (s *rsaSigner) SigInfo() (*ndn.SigConfig, error) {
	ret := &ndn.SigConfig{
		Type:    ndn.SignatureSha256WithEcdsa,
		KeyName: s.keyLocatorName,
	}
	if s.forCert {
		ret.NotBefore = utils.IdPtr(s.timer.Now())
		ret.NotAfter = utils.IdPtr(s.timer.Now().Add(s.certExpireTime))
	}
	if s.forInt {
		s.seq++
		ret.Nonce = s.timer.Nonce()
		ret.SigTime = utils.IdPtr(s.timer.Now())
		ret.SeqNum = utils.IdPtr(s.seq)
	}
	return ret, nil
}

func (s *rsaSigner) EstimateSize() uint {
	return s.keyLen
}

func (s *rsaSigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(nil, s.key, crypto.SHA256, digest)
}

// NewRsaSigner creates a signer using RSA key
func NewRsaSigner(
	forCert bool, forInt bool, expireTime time.Duration, key *rsa.PrivateKey,
	keyLocatorName enc.Name,
) ndn.Signer {
	keyLen := uint(key.Size())
	return &rsaSigner{
		timer:          basic_engine.Timer{},
		seq:            0,
		keyLocatorName: keyLocatorName,
		key:            key,
		keyLen:         keyLen,
		forCert:        forCert,
		forInt:         forInt,
		certExpireTime: expireTime,
	}
}
