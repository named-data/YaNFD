//go:generate gondn_tlv_gen
package tlv

import enc "github.com/zjkmxy/go-ndn/pkg/encoding"

type Packet struct {
	//+field:struct:Advertisement
	Advertisement *Advertisement `tlv:"0xC9"`
	//+field:struct:PrefixOpList
	PrefixOpList *PrefixOpList `tlv:"0xDD"`
}

type Advertisement struct {
	//+field:sequence:*Link:struct:Link
	Links []*Link `tlv:"0xCA"`
	//+field:sequence:*AdvEntry:struct:AdvEntry
	AdvEntries []*AdvEntry `tlv:"0xCD"`
}

type Link struct {
	//+field:natural
	Interface uint64 `tlv:"0xCB"`
	//+field:struct:Neighbor
	Neighbor *Neighbor `tlv:"0xCC"`
}

type Neighbor struct {
	//+field:name
	Name enc.Name `tlv:"0x07"`
}

type AdvEntry struct {
	//+field:struct:Destination
	Destination *Destination `tlv:"0xCE"`
	//+field:natural
	Interface uint64 `tlv:"0xCF"`
	//+field:natural
	Cost uint64 `tlv:"0xD0"`
	//+field:natural
	OtherCost uint64 `tlv:"0xD1"`
}

type Destination struct {
	//+field:name
	Name enc.Name `tlv:"0x07"`
}

type PrefixOpList struct {
	//+field:struct:Destination
	ExitRouter *Destination `tlv:"0xCE"`
	//+field:bool
	PrefixOpReset bool `tlv:"0xDE"`
	//+field:sequence:*PrefixOpAdd:struct:PrefixOpAdd
	PrefixOpAdds []*PrefixOpAdd `tlv:"0xDF"`
	//+field:sequence:*PrefixOpRemove:struct:PrefixOpRemove
	PrefixOpRemoves []*PrefixOpRemove `tlv:"0xE0"`
}

type PrefixOpAdd struct {
	//+field:name
	Name enc.Name `tlv:"0x07"`
	//+field:natural
	Cost uint64 `tlv:"0xD0"`
}

type PrefixOpRemove struct {
	//+field:name
	Name enc.Name `tlv:"0x07"`
}
