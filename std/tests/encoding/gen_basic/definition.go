//go:generate gondn_tlv_gen
package gen_basic

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type FakeMetaInfo struct {
	//+field:natural
	Number uint64 `tlv:"0x18"`
	//+field:time
	Time time.Duration `tlv:"0x19"`
	//+field:binary
	Binary []byte `tlv:"0x1a"`
}

type OptField struct {
	//+field:natural:optional
	Number *uint64 `tlv:"0x18"`
	//+field:time:optional
	Time *time.Duration `tlv:"0x19"`
	//+field:binary
	Binary []byte `tlv:"0x1a"`
	//+field:bool
	Bool bool `tlv:"0x30"`
}

type WireNameField struct {
	//+field:wire
	Wire enc.Wire `tlv:"0x01"`
	//+field:name
	Name enc.Name `tlv:"0x02"`
}
