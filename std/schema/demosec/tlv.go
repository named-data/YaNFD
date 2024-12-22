//go:generate gondn_tlv_gen
package demosec

import enc "github.com/pulsejet/ndnd/std/encoding"

// +tlv-model:nocopy
type EncryptedContent struct {
	//+field:binary
	KeyId []byte `tlv:"0x82"`
	//+field:binary
	Iv []byte `tlv:"0x84"`
	//+field:natural
	ContentLength uint64 `tlv:"0x86"`
	//+field:wire
	CipherText enc.Wire `tlv:"0x88"`
}
