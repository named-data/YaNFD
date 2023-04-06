package security

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// eccSigner is a signer that uses ECC key to sign packets.
type eccSigner struct {
	timer ndn.Timer
	seq   uint64

	keyLocatorName enc.Name
	key            *ecdsa.PrivateKey
	keyLen         uint
	forCert        bool
	forInt         bool
	certExpireTime time.Duration
}

func (s *eccSigner) SigInfo() (*ndn.SigConfig, error) {
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

func (s *eccSigner) EstimateSize() uint {
	return s.keyLen
}

func (s *eccSigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	digest := h.Sum(nil)
	return ecdsa.SignASN1(rand.Reader, s.key, digest)
}

// NewEccSigner creates a signer using ECDSA key
func NewEccSigner(
	forCert bool, forInt bool, expireTime time.Duration, key *ecdsa.PrivateKey,
	keyLocatorName enc.Name,
) ndn.Signer {
	keyLen := uint(key.Curve.Params().BitSize*2+7) / 8
	keyLen += keyLen%2 + 8
	return &eccSigner{
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
