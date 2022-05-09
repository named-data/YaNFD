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

//+tlv-model:private
type Markers struct {
	//+field:offsetMarker
	startMarker enc.PlaceHolder
	//+field:wire
	Wire enc.Wire `tlv:"0x01"`
	//+field:procedureArgument:int
	argument enc.PlaceHolder
	//+field:name
	Name enc.Name `tlv:"0x02"`
	//+field:offsetMarker
	endMarker enc.PlaceHolder
}

func (m *Markers) Encode(arg int) []byte {
	enc := MarkersEncoder{}
	enc.Init(m)
	enc.argument = arg
	wire := enc.Encode(m)
	ret := wire.Join()
	if enc.startMarker != 0 {
		return nil
	}
	if enc.endMarker != len(ret) {
		return nil
	}
	return ret
}

func ParseMarkers(buf []byte, arg int) *Markers {
	cont := MarkersParsingContext{
		argument: arg,
	}
	cont.Init()
	ret, err := cont.Parse(enc.NewBufferReader(buf), true)
	if err == nil && cont.startMarker == 0 && cont.endMarker == len(buf) {
		return ret
	} else {
		return nil
	}
}

//+tlv-model:nocopy
type NoCopyStruct struct {
	//+field:wire
	Wire1 enc.Wire `tlv:"0x01"`
	//+field:natural
	Number uint64 `tlv:"0x02"`
	//+field:wire
	Wire2 enc.Wire `tlv:"0x03"`
}
