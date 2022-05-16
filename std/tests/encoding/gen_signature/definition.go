//go:generate gondn_tlv_gen
package gen_signature

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

//+tlv-model:nocopy,private
type T1 struct {
	//+field:natural
	H1 uint64 `tlv:"1"`
	//+field:offsetMarker
	sigCoverStart enc.PlaceHolder
	//+field:natural:optional
	H2 *uint64 `tlv:"2"`
	//+field:wire
	C enc.Wire `tlv:"3"`
	//+field:signature:sigCoverStart:sigCovered
	Sig enc.Wire `tlv:"4"`

	//+field:procedureArgument:enc.Wire
	sigCovered enc.PlaceHolder
}

func (v *T1) Encode(estLen uint, value []byte) (enc.Wire, enc.Wire) {
	encoder := T1Encoder{
		Sig_estLen: estLen,
	}
	encoder.Init(v)
	wire := encoder.Encode(v)
	if encoder.Sig_wireIdx >= 0 {
		wire[encoder.Sig_wireIdx] = value
		// Fix length, assuming len(value) <= estLen < 253
		buf := wire[encoder.Sig_wireIdx-1]
		buf[len(buf)-1] = byte(len(value))
	}

	return wire, encoder.sigCovered
}

func ReadT1(reader enc.ParseReader) (*T1, enc.Wire, error) {
	context := T1ParsingContext{}
	context.Init()
	ret, err := context.Parse(reader, false)
	if err != nil {
		return nil, nil, err
	}
	return ret, context.sigCovered, nil
}
