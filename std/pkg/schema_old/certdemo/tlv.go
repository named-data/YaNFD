//go:generate gondn_tlv_gen
package certdemo

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type CaProfile struct {
	//+field:name
	CaPrefix enc.Name `tlv:"0x81"`
	//+field:string
	CaInfo string `tlv:"0x83"`
	//+field:sequence:string:string
	ParamKey []string `tlv:"0x85"`
	//+field:natural
	MaxValidPeriod uint64 `tlv:"0x8B"`
	//+field:wire
	CaCert enc.Wire `tlv:"0x89"`
}

// The original definition in the spec cannot be organized into a struct
// So I modified it
type Param struct {
	//+field:string
	ParamKey string `tlv:"0x85"`
	//+field:binary
	ParamValue []byte `tlv:"0x87"`
}

type ProbeInt struct {
	//+field:sequence:*Param:struct:Param
	Params []*Param `tlv:"0xC1"`
}

type ProbeRes struct {
	//+field:name
	Response enc.Name `tlv:"0x8D"`
	//+field:natural:optional
	MaxSuffixLength *uint64
}

type CmdNewInt struct {
	//+field:binary
	EcdhPub []byte `tlv:"0x91"`
	//+field:binary
	CertReq []byte `tlv:"0x93"`
}

type CmdNewData struct {
	//+field:binary
	EcdhPub []byte `tlv:"0x91"`
	//+field:binary
	Salt []byte `tlv:"0x95"`
	//+field:binary
	ReqId []byte `tlv:"0x97"`
	//+field:sequence:string:string
	Challenge []string `tlv:"0x99"`
}

type CipherMsg struct {
	//+field:binary
	InitVec []byte `tlv:"0x9D"`
	//+field:binary
	AuthNTag []byte `tlv:"0xAF"`
	//+field:binary
	Payload []byte `tlv:"0x9F"`
}

type ChallengeIntPlain struct {
	//+field:string
	SelectedChal string `tlv:"0xA1"`
	//+field:sequence:*Param:struct:Param
	Params []*Param `tlv:"0xC1"`
}

type ChallengeDataPlain struct {
	//+field:natural
	Status uint64 `tlv:"0x9B"`
	//+field:natural:optional
	ChalStatus *uint64 `tlv:"0xA3"`
	//+field:natural:optional
	RemainTries *uint64 `tlv:"0xA5"`
	//+field:natural:optional
	RemainTime *uint64 `tlv:"0xA7"`
	//+field:name
	CertName enc.Name `tlv:"0xA9"`
	//+field:name
	ForwardingHint enc.Name `tlv:"0x1e"`
	//+field:sequence:*Param:struct:Param
	Params []*Param `tlv:"0xC1"`
}
