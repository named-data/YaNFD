//go:generate gondn_tlv_gen
package svs

type StateVecEntry struct {
	//+field:binary
	NodeId []byte `tlv:"0x07"`
	//+field:natural
	SeqNo uint64 `tlv:"0xcc"`
}

type StateVec struct {
	//+field:sequence:*StateVecEntry:struct:StateVecEntry
	Entries []*StateVecEntry `tlv:"0xca"`
}

type StateVecAppParam struct {
	//+field:struct:StateVec
	Entries []*StateVec `tlv:"0xc9"`
}
