// Basic policies for test and demo use
package schema

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

type RegisterPolicy struct{}

func (p *RegisterPolicy) PolicyTrait() NTPolicy {
	return p
}

func (*RegisterPolicy) onAttach(path enc.NamePattern, engine ndn.Engine) error {
	prefix := make(enc.Name, len(path))
	for i, p := range path {
		c, ok := p.(enc.Component)
		if !ok {
			return errors.New("unable to register a prefix that has unknown pattern")
		}
		prefix[i] = c
	}
	return engine.RegisterRoute(prefix)
}

func (p *RegisterPolicy) Apply(node NTNode) error {
	return AddEventListener(node, PropOnAttach, p.onAttach)
}

func NewRegisterPolicy() NTPolicy {
	return &RegisterPolicy{}
}

type LocalOnlyPolicy struct{}

func (p *LocalOnlyPolicy) PolicyTrait() NTPolicy {
	return p
}

func (p *LocalOnlyPolicy) Apply(node NTNode) error {
	node.Set(PropSuppressInt, true) // Ignore error on non-expressible points
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func NewLocalOnlyPolicy() NTPolicy {
	return &LocalOnlyPolicy{}
}

type CacheEntry struct {
	RawData  enc.Wire
	Validity time.Time
}

type MemStoragePolicy struct {
	timer ndn.Timer
	tree  *basic_engine.NameTrie[CacheEntry]
}

func (p *MemStoragePolicy) PolicyTrait() NTPolicy {
	return p
}

func (p *MemStoragePolicy) Get(name enc.Name, canBePrefix bool, mustBeFresh bool) enc.Wire {
	node := p.tree.ExactMatch(name)
	now := time.Time{}
	if p.timer != nil {
		now = p.timer.Now()
	}
	if node == nil {
		return nil
	}
	freshTest := func(entry CacheEntry) bool {
		return len(entry.RawData) > 0 && (!mustBeFresh || entry.Validity.After(now))
	}
	if freshTest(node.Value()) {
		return node.Value().RawData
	}
	dataNode := node.FirstNodeIf(freshTest)
	if dataNode != nil {
		return dataNode.Value().RawData
	} else {
		return nil
	}
}

func (p *MemStoragePolicy) Put(name enc.Name, rawData enc.Wire, validity time.Time) {
	node := p.tree.MatchAlways(name)
	node.SetValue(CacheEntry{
		RawData:  rawData,
		Validity: validity,
	})
}

func (p *MemStoragePolicy) onAttach(path enc.NamePattern, engine ndn.Engine) error {
	p.timer = engine.Timer()
	return nil
}

func (p *MemStoragePolicy) onSearch(
	matching enc.Matching, name enc.Name, canBePrefix bool, mustBeFresh bool, context Context,
) enc.Wire {
	return p.Get(name, canBePrefix, mustBeFresh)
}

func (p *MemStoragePolicy) onSave(
	matching enc.Matching, name enc.Name, rawData enc.Wire, validity time.Time, context Context,
) {
	p.Put(name, rawData, validity)
}

func (p *MemStoragePolicy) Apply(node NTNode) error {
	AddEventListener(node, PropOnAttach, p.onAttach)
	AddEventListener(node, PropOnSearchStorage, p.onSearch)
	AddEventListener(node, PropOnSaveStorage, p.onSave)
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func NewMemStoragePolicy() NTPolicy {
	return &MemStoragePolicy{
		tree: basic_engine.NewNameTrie[CacheEntry](),
	}
}

// FixedKeySigner is a demo policy that signs data using provided HMAC key.
type FixedKeySigner struct {
	key []byte
}

func (p *FixedKeySigner) PolicyTrait() NTPolicy {
	return p
}

func (*FixedKeySigner) SigInfo() (*ndn.SigConfig, error) {
	return &ndn.SigConfig{
		Type:    ndn.SignatureHmacWithSha256,
		KeyName: nil,
	}, nil
}

func (*FixedKeySigner) EstimateSize() uint {
	return 32
}

func (p *FixedKeySigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	mac := hmac.New(sha256.New, p.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return mac.Sum(nil), nil
}

func (p *FixedKeySigner) onValidateData(
	_ enc.Matching, _ enc.Name, sig ndn.Signature, covered enc.Wire, _ Context,
) ValidRes {
	if sig.SigType() != ndn.SignatureHmacWithSha256 {
		return VrFail
	}
	mac := hmac.New(sha256.New, p.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return VrFail
		}
	}
	if hmac.Equal(mac.Sum(nil), sig.SigValue()) {
		return VrPass
	} else {
		return VrFail
	}
}

func (p *FixedKeySigner) Apply(node NTNode) error {
	AddEventListener(node, PropOnValidateData, p.onValidateData)
	node.Set(PropDataSigner, ndn.Signer(p))
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func NewFixedKeySigner(key []byte) NTPolicy {
	return &FixedKeySigner{
		key: key,
	}
}
