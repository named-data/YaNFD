//go:generate gondn_tlv_gen
package crdt

type IDType struct {
	//+field:natural
	Producer uint64 `tlv:"0xa1"`
	//+field:natural
	Clock uint64 `tlv:"0xa3"`
}

const (
	RecordNone uint64 = iota
	RecordInsert
	RecordDelete
)

type Record struct {
	//+field:natural
	RecordType uint64 `tlv:"0xa5"`
	//+field:struct:IDType
	ID *IDType `tlv:"0xa7"`
	//+field:struct:IDType
	Origin *IDType `tlv:"0xa9"`
	//+field:struct:IDType
	RightOrigin *IDType `tlv:"0xaa"`
	//+field:string
	Content string `tlv:"0xac"`
}
