//go:generate gondn_tlv_gen
package ndncert_0_3

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

type ProbeIntAppParam struct {
	//+field:map:string:string:0x87:[]byte:binary
	Params map[string][]byte `tlv:"0x85"`
}

type ProbeRes struct {
	//+field:name
	Response enc.Name `tlv:"0x07"`
	//+field:natural:optional
	MaxSuffixLength *uint64
}

type ProbeResContent struct {
	//+field:sequence:*ProbeRes:struct:ProbeRes
	Vals []*ProbeRes `tlv:"0x8D"`
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
	//+field:map:string:string:0x87:[]byte:binary
	Params map[string][]byte `tlv:"0x85"`
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
	//+field:map:string:string:0x87:[]byte:binary
	Params map[string][]byte `tlv:"0x85"`
}

type ErrorMsgData struct {
	//+field:natural
	ErrCode uint64 `tlv:"0xAB"`
	//+field:string
	ErrInfo string `tlv:"0xAD"`
}
