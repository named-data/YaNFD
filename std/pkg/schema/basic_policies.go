package schema

import (
	"errors"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
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
	prop := node.Get(PropOnAttach)
	if prop == nil {
		return errors.New("policy Register: specified node does not have OnAttach event")
	}
	evt, ok := prop.(*Event[*NodeOnAttachEvent])
	if !ok || evt == nil {
		return errors.New("policy Register: specified node does not have OnAttach event")
	}
	evt.Add(utils.IdPtr(p.onAttach))
	return nil
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

func (p *MemStoragePolicy) onSearch(matching enc.Matching, name enc.Name, context Context) enc.Wire {
	interest, ok := context[CkInterest].(ndn.Interest)
	if !ok || interest == nil {
		return nil
	}
	return p.Get(name, interest.CanBePrefix(), interest.MustBeFresh())
}

func (p *MemStoragePolicy) onSave(
	matching enc.Matching, name enc.Name, rawData enc.Wire, validity time.Time, context Context,
) {
	p.Put(name, rawData, validity)
}

func (p *MemStoragePolicy) Apply(node NTNode) error {
	attachEvt, ok := node.Get(PropOnAttach).(*Event[*NodeOnAttachEvent])
	if ok && attachEvt != nil {
		attachEvt.Add(utils.IdPtr(p.onAttach))
	}
	searchEvt, ok := node.Get(PropOnSearchStorage).(*Event[*NodeSearchStorageEvent])
	if ok && searchEvt != nil {
		searchEvt.Add(utils.IdPtr(p.onSearch))
	}
	saveEvt, ok := node.Get(PropOnSaveStorage).(*Event[*NodeSaveStorageEvent])
	if ok && searchEvt != nil {
		saveEvt.Add(utils.IdPtr(p.onSave))
	}
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