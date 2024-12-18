package mgmt

import (
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/table"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	ndn_mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type ReadvertiseNlsr struct {
	m *Thread
}

func (r *ReadvertiseNlsr) String() string {
	return "ReadvertiseNlsr"
}

func (r *ReadvertiseNlsr) Announce(name enc.Name, route *table.Route) {
	if route.Origin != table.RouteOriginClient {
		core.LogDebug(r, "skip advertise=", name, " origin=", route.Origin)
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

func (r *ReadvertiseNlsr) Withdraw(name enc.Name, faceID uint64, origin uint64) {
	if origin != table.RouteOriginClient {
		core.LogDebug(r, "skip withdraw=", name, " origin=", origin)
		return
	}
	core.LogInfo(r, "withdraw=", name)

	params := &ndn_mgmt.ControlArgs{
		Name:   name,
		FaceId: utils.IdPtr(faceID),
		Origin: utils.IdPtr(origin),
	}

	iparams := &ndn_mgmt.ControlParameters{
		Val: &ndn_mgmt.ControlArgs{Name: name},
	}
	cmd, _ := enc.NameFromStr("/localhost/nlsr/rib/unregister")
	cmd = append(cmd, enc.NewBytesComponent(enc.TypeGenericNameComponent, iparams.Encode().Join()))

	r.m.sendInterest(cmd, params.Encode())
}
