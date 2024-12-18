package mgmt

import (
	"sync"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	ndn_mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// Simple readvertiser that echoes the register command to NLSR.
// Currently the command is one-shot, and does not handle failures.
type NlsrReadvertiser struct {
	m *Thread
	// List of routes already advertised to NLSR
	advertised map[uint64]int // hash -> count
	// This is called from RIB (i.e. could be fw threads)
	mutex sync.Mutex
}

func NewNlsrReadvertiser(m *Thread) *NlsrReadvertiser {
	return &NlsrReadvertiser{
		m:          m,
		advertised: make(map[uint64]int),
	}
}

func (r *NlsrReadvertiser) String() string {
	return "NlsrReadvertiser"
}

func (r *NlsrReadvertiser) Announce(name enc.Name, route *table.Route) {
	if route.Origin != table.RouteOriginClient {
		core.LogDebug(r, "skip advertise=", name, " origin=", route.Origin)
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	nhash := name.Hash()
	r.advertised[nhash] += 1
	if r.advertised[nhash] > 1 {
		core.LogDebug(r, "skip advertise=", name, " already advertised")
		return
	}
	core.LogInfo(r, "advertise=", name)

	params := &ndn_mgmt.ControlArgs{
		Name:   name,
		FaceId: utils.IdPtr(route.FaceID),
		Origin: utils.IdPtr(route.Origin),
		Cost:   utils.IdPtr(route.Cost),
		Flags:  utils.IdPtr(route.Flags),
	}

	iparams := &ndn_mgmt.ControlParameters{
		Val: &ndn_mgmt.ControlArgs{Name: name},
	}
	cmd, _ := enc.NameFromStr("/localhost/nlsr/rib/register")
	cmd = append(cmd, enc.NewBytesComponent(enc.TypeGenericNameComponent, iparams.Encode().Join()))

	r.m.sendInterest(cmd, params.Encode())
}

func (r *NlsrReadvertiser) Withdraw(name enc.Name, route *table.Route) {
	if route.Origin != table.RouteOriginClient {
		core.LogDebug(r, "skip withdraw=", name, " origin=", route.Origin)
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	nhash := name.Hash()
	r.advertised[nhash] -= 1
	if r.advertised[nhash] > 0 {
		core.LogDebug(r, "skip withdraw=", name, " still advertised")
		return
	}
	core.LogInfo(r, "withdraw=", name)

	params := &ndn_mgmt.ControlArgs{
		Name:   name,
		FaceId: utils.IdPtr(route.FaceID),
		Origin: utils.IdPtr(route.Origin),
	}

	iparams := &ndn_mgmt.ControlParameters{
		Val: &ndn_mgmt.ControlArgs{Name: name},
	}
	cmd, _ := enc.NameFromStr("/localhost/nlsr/rib/unregister")
	cmd = append(cmd, enc.NewBytesComponent(enc.TypeGenericNameComponent, iparams.Encode().Join()))

	r.m.sendInterest(cmd, params.Encode())
}
