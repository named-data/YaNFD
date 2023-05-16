//go:generate gondn_tlv_gen
package gen_map

type StringMap struct {
	//+field:map:string:string:0x87:[]byte:binary
	Params map[string][]byte `tlv:"0x85"`
}

type Inner struct {
	//+field:natural
	Num uint64 `tlv:"0x01"`
}

type IntStructMap struct {
	//+field:map:uint64:natural:0x87:*Inner:struct:Inner
	Params map[uint64]*Inner `tlv:"0x85"`
}
