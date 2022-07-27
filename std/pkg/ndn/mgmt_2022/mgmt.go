package mgmt_2022

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

type MgmtConfig struct {
	// local means whether NFD is of localhost
	local bool
	// signer is the signer used to sign the command
	signer ndn.Signer
	// spec is the NDN spec used to make Interests
	spec ndn.Spec
}

// MakeCmd makes and encodes a NFD mgmt command Interest.
// Currently NFD and YaNFD supports signed Interest.
func (mgmt *MgmtConfig) MakeCmd(module string, cmd string, args *ControlArgs,
	intParam *ndn.InterestConfig) (enc.Name, enc.Wire, error) {

	var err error = nil
	val := ControlParameters{
		Val: args,
	}

	// Make first part of name
	name := enc.Name(nil)
	if mgmt.local {
		name, err = enc.NameFromStr("/localhost/nfd/" + module + "/" + cmd)
	} else {
		name, err = enc.NameFromStr("/localhop/nfd/" + module + "/" + cmd)
	}
	if err != nil {
		return nil, nil, err
	}
	name = append(name, enc.NewBytesComponent(enc.TypeGenericNameComponent, val.Bytes()))

	// Make and sign Interest
	wire, _, finalName, err := mgmt.spec.MakeInterest(name, intParam, enc.Wire{}, mgmt.signer)
	if err != nil {
		return nil, nil, err
	}

	return finalName, wire, nil
}

// MakeCmdDict is the same as MakeCmd but receives a map[string]any as arguments.
func (mgmt *MgmtConfig) MakeCmdDict(module string, cmd string, args map[string]any,
	intParam *ndn.InterestConfig) (enc.Name, enc.Wire, error) {
	// Parse arguments
	vv, err := DictToControlArgs(args)
	if err != nil {
		return nil, nil, err
	}
	return mgmt.MakeCmd(module, cmd, vv, intParam)
}
