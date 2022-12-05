// Basic policies for test and demo use. Not secure for production.
// The TODO points listed here are all design questions we need to decide before production-ready code.
package demo

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// RegisterPolicy marks the current node as the prefix to be registered.
// The current one can only handle fixed prefix.
// TODO: Handle the path "/prefix/<node-id>" with a given <node-id>. (#ENV)
type RegisterPolicy struct{}

func (p *RegisterPolicy) PolicyTrait() schema.NTPolicy {
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

func (p *RegisterPolicy) Apply(node schema.NTNode) error {
	return schema.AddEventListener(node, schema.PropOnAttach, p.onAttach)
}

func NewRegisterPolicy() schema.NTPolicy {
	return &RegisterPolicy{}
}

// LocalOnlyPolicy surpress Interest expression.
// TODO: Is this secure? Do we need to consider the case where PropSuppressInt is overwritten by another policy?
type LocalOnlyPolicy struct{}

func (p *LocalOnlyPolicy) PolicyTrait() schema.NTPolicy {
	return p
}

func (p *LocalOnlyPolicy) Apply(node schema.NTNode) error {
	node.Set(schema.PropSuppressInt, true) // Ignore error on non-expressible points
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func NewLocalOnlyPolicy() schema.NTPolicy {
	return &LocalOnlyPolicy{}
}

type CacheEntry struct {
	RawData  enc.Wire
	Validity time.Time
}

// MemStoragePolicy is a policy that stored data in a memory storage.
// TODO: If we use on-disk storage, how to specify the path (#ENV)
type MemStoragePolicy struct {
	timer ndn.Timer
	lock  sync.RWMutex
	tree  *basic_engine.NameTrie[CacheEntry]
}

func (p *MemStoragePolicy) PolicyTrait() schema.NTPolicy {
	return p
}

func (p *MemStoragePolicy) Get(name enc.Name, canBePrefix bool, mustBeFresh bool) enc.Wire {
	p.lock.RLock()
	defer p.lock.RUnlock()

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
	p.lock.Lock()
	defer p.lock.Unlock()

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
	matching enc.Matching, name enc.Name, canBePrefix bool, mustBeFresh bool, context schema.Context,
) enc.Wire {
	return p.Get(name, canBePrefix, mustBeFresh)
}

func (p *MemStoragePolicy) onSave(
	matching enc.Matching, name enc.Name, rawData enc.Wire, validity time.Time, context schema.Context,
) {
	p.Put(name, rawData, validity)
}

func (p *MemStoragePolicy) Apply(node schema.NTNode) error {
	schema.AddEventListener(node, schema.PropOnAttach, p.onAttach)
	schema.AddEventListener(node, schema.PropOnSearchStorage, p.onSearch)
	schema.AddEventListener(node, schema.PropOnSaveStorage, p.onSave)
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func NewMemStoragePolicy() schema.NTPolicy {
	return &MemStoragePolicy{
		tree: basic_engine.NewNameTrie[CacheEntry](),
	}
}

// FixedKeySigner is a demo policy that signs data using provided HMAC key.
// TODO: This has a problem with group signature node:
// The group signature node (subtree) has two leaves: the segmented data, and meta data.
// The segmented data has its own validation (SHA256 sig), but how to validate the meta data is specified by
// the trust schema (i.e. user). Then, is it still a good idea to make group sig node a blackbox?
// If yes, what is the best way to let the user specify how the packet is signed/validated? (#BLACKBOX)
type FixedKeySigner struct {
	key []byte
}

func (p *FixedKeySigner) PolicyTrait() schema.NTPolicy {
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
	_ enc.Matching, _ enc.Name, sig ndn.Signature, covered enc.Wire, _ schema.Context,
) schema.ValidRes {
	if sig.SigType() != ndn.SignatureHmacWithSha256 {
		return schema.VrFail
	}
	mac := hmac.New(sha256.New, p.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return schema.VrFail
		}
	}
	if hmac.Equal(mac.Sum(nil), sig.SigValue()) {
		return schema.VrPass
	} else {
		return schema.VrFail
	}
}

func (p *FixedKeySigner) Apply(node schema.NTNode) error {
	schema.AddEventListener(node, schema.PropOnValidateData, p.onValidateData)
	node.Set(schema.PropDataSigner, ndn.Signer(p))
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func NewFixedKeySigner(key []byte) schema.NTPolicy {
	return &FixedKeySigner{
		key: key,
	}
}

// FixedKeyIntSigner is a demo policy that signs Interests using provided HMAC key.
type FixedKeyIntSigner struct {
	key   []byte
	timer ndn.Timer
	seq   uint64
}

func (p *FixedKeyIntSigner) PolicyTrait() schema.NTPolicy {
	return p
}

func (p *FixedKeyIntSigner) SigInfo() (*ndn.SigConfig, error) {
	p.seq++
	return &ndn.SigConfig{
		Type:    ndn.SignatureHmacWithSha256,
		KeyName: enc.Name{enc.Component{Typ: enc.TypeGenericNameComponent, Val: p.key}},
		Nonce:   p.timer.Nonce(),
		SigTime: utils.IdPtr(p.timer.Now()),
		SeqNum:  utils.IdPtr(p.seq),
	}, nil
}

func (*FixedKeyIntSigner) EstimateSize() uint {
	return 32
}

func (p *FixedKeyIntSigner) ComputeSigValue(covered enc.Wire) ([]byte, error) {
	mac := hmac.New(sha256.New, p.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return mac.Sum(nil), nil
}

func (p *FixedKeyIntSigner) onValidateInt(
	_ enc.Matching, _ enc.Name, sig ndn.Signature, covered enc.Wire, _ schema.Context,
) schema.ValidRes {
	if sig.SigType() != ndn.SignatureHmacWithSha256 {
		return schema.VrFail
	}
	mac := hmac.New(sha256.New, p.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return schema.VrFail
		}
	}
	if hmac.Equal(mac.Sum(nil), sig.SigValue()) {
		return schema.VrPass
	} else {
		return schema.VrFail
	}
}

func (p *FixedKeyIntSigner) Apply(node schema.NTNode) error {
	// TODO: Does not look good but I cannot come up with a better design
	// This is to avoid leaf nodes, as they represent not-on-demand-produced data
	if _, ok := node.(*schema.LeafNode); ok {
		return nil
	}
	schema.AddEventListener(node, schema.PropOnValidateInt, p.onValidateInt)
	node.Set(schema.PropIntSigner, ndn.Signer(p))
	schema.AddEventListener(node, schema.PropOnAttach, p.onAttach)
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
	return nil
}

func (p *FixedKeyIntSigner) onAttach(path enc.NamePattern, engine ndn.Engine) error {
	p.timer = engine.Timer()
	p.seq = 0
	return nil
}

func NewFixedKeyIntSigner(key []byte) schema.NTPolicy {
	return &FixedKeyIntSigner{
		key: key,
	}
}
