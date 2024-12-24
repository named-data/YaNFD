//go:generate gondn_tlv_gen
package gen_composition

import (
	enc "github.com/named-data/ndnd/std/encoding"
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

// +tlv-model:nocopy,private
type InnerWire1 struct {
	//+field:wire
	Wire1 enc.Wire `tlv:"0x01"`
	//+field:natural:optional
	Num *uint64 `tlv:"0x02"`
}

// +tlv-model:nocopy,private
type InnerWire2 struct {
	//+field:wire
	Wire2 enc.Wire `tlv:"0x03"`
}

// +tlv-model:nocopy
type NestedWire struct {
	//+field:struct:InnerWire1:nocopy
	W1 *InnerWire1 `tlv:"0x04"`
	//+field:natural
	N uint64 `tlv:"0x05"`
	//+field:struct:InnerWire2:nocopy
	W2 *InnerWire2 `tlv:"0x06"`
}
