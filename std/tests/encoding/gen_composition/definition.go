//go:generate gondn_tlv_gen
package gen_composition

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type IntArray struct {
	//+field:sequence:uint64:natural
	Words []uint64 `tlv:"0x01"`
}

type NameArray struct {
	//+field:sequence:enc.Name:name
	Names []enc.Name `tlv:"0x07"`
}

type Inner struct {
	//+field:natural
	Num uint64 `tlv:"0x01"`
}

type Nested struct {
	//+field:struct:Inner
	Val *Inner `tlv:"0x02"`
}

type NestedSeq struct {
	//+field:sequence:*Inner:struct:Inner
	Vals []*Inner `tlv:"0x03"`
}
