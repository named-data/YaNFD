package schema

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	basic_engine "github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/ndn"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
)

type RegisterPolicy struct {
	RegisterIf bool
	// It is map[string]any in json
	// but the any can be a string
	Patterns enc.Matching
}

func (p *RegisterPolicy) PolicyTrait() Policy {
	return p
}

func (p *RegisterPolicy) onAttach(event *Event) any {
	node := event.TargetNode
	mNode := node.Apply(p.Patterns)
	if mNode == nil {
		panic("cannot initialize the name prefix to register")
	}
	err := node.Engine().RegisterRoute(mNode.Name)
	if err != nil {
		panic(fmt.Errorf("prefix registration failed: %+v", err))
	}
	return nil
}

func (p *RegisterPolicy) Apply(node *Node) {
	if p.RegisterIf {
		var callback Callback = p.onAttach
		node.AddEventListener(PropOnAttach, &callback)
	}
}

func NewRegisterPolicy() Policy {
	return &RegisterPolicy{
		RegisterIf: true,
	}
}

type Sha256SignerPolicy struct{}

func (p *Sha256SignerPolicy) PolicyTrait() Policy {
	return p
}

func NewSha256SignerPolicy() Policy {
	return &Sha256SignerPolicy{}
}

func (p *Sha256SignerPolicy) onGetDataSigner(*Event) any {
	return sec.NewSha256Signer()
}

func (p *Sha256SignerPolicy) onValidateData(event *Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureDigestSha256 {
		return VrSilence
	}
	val, _ := sec.NewSha256Signer().ComputeSigValue(sigCovered)
	if bytes.Equal(signature.SigValue(), val) {
		return VrPass
	} else {
		return VrFail
	}
}

func (p *Sha256SignerPolicy) Apply(node *Node) {
	// IdPtr must be used
	evt := node.GetEvent(PropOnGetDataSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetDataSigner))
	}
	// PropOnValidateData must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(PropOnValidateData)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateData))
	} else {
		panic("attaching Sha256SignerPolicy to a node that does not need to validate Data. What is the use?")
	}
}

type CacheEntry struct {
	RawData  enc.Wire
	Validity time.Time
}

// MemStoragePolicy is a policy that stored data in a memory storage.
// It will iteratively applies to all children in a subtree.
type MemStoragePolicy struct {
	timer ndn.Timer
	lock  sync.RWMutex
	// TODO: A better implementation would be MemStoragePolicy refers to an external storage
	// but not implement one itself.
	tree *basic_engine.NameTrie[CacheEntry]
}

func (p *MemStoragePolicy) PolicyTrait() Policy {
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

func (p *MemStoragePolicy) onAttach(event *Event) any {
	p.timer = event.TargetNode.Engine().Timer()
	return nil
}

func (p *MemStoragePolicy) onSearch(event *Event) any {
	// event.IntConfig is always valid for onSearch, no matter if there is an Interest.
	return p.Get(event.Target.Name, event.IntConfig.CanBePrefix, event.IntConfig.MustBeFresh)
}

func (p *MemStoragePolicy) onSave(event *Event) any {
	validity := p.timer.Now().Add(*event.ValidDuration)
	p.Put(event.Target.Name, event.RawPacket, validity)
	return nil
}

func (p *MemStoragePolicy) Apply(node *Node) {
	// TODO: onAttach does not need to be called on every child...
	// But I don't have enough time to fix this
	if event := node.GetEvent(PropOnAttach); event != nil {
		event.Add(utils.IdPtr(p.onAttach))
	}
	if event := node.GetEvent(PropOnSearchStorage); event != nil {
		event.Add(utils.IdPtr(p.onSearch))
	}
	if event := node.GetEvent(PropOnSaveStorage); event != nil {
		event.Add(utils.IdPtr(p.onSave))
	}
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
}

func NewMemStoragePolicy() Policy {
	return &MemStoragePolicy{
		tree: basic_engine.NewNameTrie[CacheEntry](),
	}
}

type FixedHmacSignerPolicy struct {
	Key         string
	KeyName     enc.Name
	SignForCert bool
	ExpireTime  time.Duration
}

func (p *FixedHmacSignerPolicy) PolicyTrait() Policy {
	return p
}

func NewFixedHmacSignerPolicy() Policy {
	return &FixedHmacSignerPolicy{
		SignForCert: false,
	}
}

func (p *FixedHmacSignerPolicy) onGetDataSigner(*Event) any {
	return sec.NewHmacSigner(p.KeyName, []byte(p.Key), p.SignForCert, p.ExpireTime)
}

func (p *FixedHmacSignerPolicy) onValidateData(event *Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureHmacWithSha256 {
		return VrSilence
	}
	if sec.CheckHmacSig(sigCovered, signature.SigValue(), []byte(p.Key)) {
		return VrPass
	} else {
		return VrFail
	}
}

func (p *FixedHmacSignerPolicy) Apply(node *Node) {
	// key must present
	if len(p.Key) == 0 {
		panic("FixedHmacSignerPolicy requires key to present before apply.")
	}
	// IdPtr must be used
	evt := node.GetEvent(PropOnGetDataSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetDataSigner))
	}
	// PropOnValidateData must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(PropOnValidateData)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateData))
	} else {
		panic("applying FixedHmacSignerPolicy to a node that does not need to validate Data. What is the use?")
	}
}

type FixedHmacIntSignerPolicy struct {
	Key    string
	signer ndn.Signer
}

func (p *FixedHmacIntSignerPolicy) PolicyTrait() Policy {
	return p
}

func NewFixedHmacIntSignerPolicy() Policy {
	return &FixedHmacIntSignerPolicy{}
}

func (p *FixedHmacIntSignerPolicy) onGetIntSigner(*Event) any {
	return p.signer
}

func (p *FixedHmacIntSignerPolicy) onValidateInt(event *Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureHmacWithSha256 {
		return VrSilence
	}
	if sec.CheckHmacSig(sigCovered, signature.SigValue(), []byte(p.Key)) {
		return VrPass
	} else {
		return VrFail
	}
}

func (p *FixedHmacIntSignerPolicy) onAttach(event *Event) any {
	p.signer = sec.NewHmacIntSigner([]byte(p.Key), event.TargetNode.Engine().Timer())
	return nil
}

func (p *FixedHmacIntSignerPolicy) Apply(node *Node) {
	// key must present
	if len(p.Key) == 0 {
		panic("FixedHmacSignerPolicy requires key to present before apply.")
	}
	// IdPtr must be used
	evt := node.GetEvent(PropOnGetIntSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetIntSigner))
	}
	// PropOnValidateInt must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(PropOnValidateInt)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateInt))
	} else {
		panic("applying FixedHmacSignerPolicy to a node that does not need to validate Interest. What is the use?")
	}

	node.AddEventListener(PropOnAttach, utils.IdPtr(p.onAttach))
}

func initPolicies() {
	registerPolicyDesc := &PolicyImplDesc{
		ClassName: "RegisterPolicy",
		Properties: map[PropKey]PropertyDesc{
			"RegisterIf": DefaultPropertyDesc("RegisterIf"),
			"Patterns":   MatchingPropertyDesc("Patterns"),
		},
		Create: NewRegisterPolicy,
	}
	sha256SignerPolicyDesc := &PolicyImplDesc{
		ClassName: "Sha256Signer",
		Create:    NewSha256SignerPolicy,
	}
	RegisterPolicyImpl(registerPolicyDesc)
	RegisterPolicyImpl(sha256SignerPolicyDesc)
	memoryStoragePolicyDesc := &PolicyImplDesc{
		ClassName: "MemStorage",
		Create:    NewMemStoragePolicy,
	}
	RegisterPolicyImpl(memoryStoragePolicyDesc)

	fixedHmacSignerPolicyDesc := &PolicyImplDesc{
		ClassName: "FixedHmacSigner",
		Create:    NewFixedHmacSignerPolicy,
		Properties: map[PropKey]PropertyDesc{
			"KeyValue":    DefaultPropertyDesc("Key"),
			"KeyName":     NamePropertyDesc("KeyName"),
			"SignForCert": DefaultPropertyDesc("SignForCert"),
			"ExpireTime":  TimePropertyDesc("ExpireTime"),
		},
	}
	RegisterPolicyImpl(fixedHmacSignerPolicyDesc)

	fixedHmacIntSignerPolicyDesc := &PolicyImplDesc{
		ClassName: "FixedHmacIntSigner",
		Create:    NewFixedHmacIntSignerPolicy,
		Properties: map[PropKey]PropertyDesc{
			"KeyValue": DefaultPropertyDesc("Key"),
		},
	}
	RegisterPolicyImpl(fixedHmacIntSignerPolicyDesc)
}
