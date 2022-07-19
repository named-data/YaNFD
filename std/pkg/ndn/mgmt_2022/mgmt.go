package mgmt_2022

import (
	"errors"
	"math/rand"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	spec "github.com/zjkmxy/go-ndn/pkg/ndn/spec_2022"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

type MgmtConfig struct {
	// local means whether NFD is of localhost
	local bool
	// signer is the signer used to sign the command
	signer ndn.Signer
	// timer is used to generate timestamp
	timer ndn.Timer
}

// MakeCmd makes a NFD mgmt command Interest Name.
// Currently NFD does not use AppParam, so it only returns a command name.
func (mgmt *MgmtConfig) MakeCmd(module string, cmd string, args map[string]any) (enc.Name, error) {
	// Parse arguments
	vv, err := DictToControlArgs(args)
	if err != nil {
		return nil, err
	}
	val := ControlParameters{
		Val: vv,
	}

	// Make first part of name
	name := enc.Name(nil)
	if mgmt.local {
		name, err = enc.NameFromStr("/localhost/nfd/" + module + "/" + cmd)
	} else {
		name, err = enc.NameFromStr("/localhop/nfd/" + module + "/" + cmd)
	}
	if err != nil {
		return nil, err
	}
	name = append(name, enc.NewBytesComponent(enc.TypeGenericNameComponent, val.Bytes()))

	// Timestamp and nonce
	tim := utils.MakeTimestamp(mgmt.timer.Now())
	name = append(name, enc.NewNumberComponent(enc.TypeGenericNameComponent, tim))
	nonce := rand.Uint64()
	name = append(name, enc.NewNumberComponent(enc.TypeGenericNameComponent, nonce))

	// SignatureInfo
	sigInfo, err := mgmt.signer.SigInfo()
	if err != nil {
		return nil, err
	}
	sigInfoBytes, err := spec.Spec{}.EncodeSigInfo(sigInfo)
	if err != nil {
		return nil, err
	}
	if len(sigInfoBytes) > 253 {
		return nil, errors.New("SignatureInfo is too long")
	}
	siComp := []byte{0x2c, byte(len(sigInfoBytes))}
	siComp = append(siComp, sigInfoBytes...)
	name = append(name, enc.NewBytesComponent(enc.TypeGenericNameComponent, siComp))

	// SignatureValue
	bufToSign := make(enc.Wire, len(name))
	for i, c := range name {
		bufToSign[i] = c.Bytes()
	}
	sigValue, err := mgmt.signer.ComputeSigValue(bufToSign)
	if err != nil {
		return nil, err
	}
	if len(sigValue) > 253 {
		return nil, errors.New("SignatureValue is too long")
	}
	svComp := []byte{0x2e, byte(len(sigValue))}
	svComp = append(svComp, sigValue...)
	name = append(name, enc.NewBytesComponent(enc.TypeGenericNameComponent, svComp))

	return name, nil
}
