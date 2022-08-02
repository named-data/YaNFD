//go:generate gondn_tlv_gen
package spec_2022

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type KeyLocator struct {
	//+field:name
	Name enc.Name `tlv:"0x07"`
	//+field:binary
	KeyDigest []byte `tlv:"0x1d"`
}

type Links struct {
	//+field:sequence:enc.Name:name
	Names []enc.Name `tlv:"0x07"`
}

type MetaInfo struct {
	//+field:natural:optional
	ContentType *uint64 `tlv:"0x18"`
	//+field:time:optional
	FreshnessPeriod *time.Duration `tlv:"0x19"`
	//+field:binary
	FinalBlockID []byte `tlv:"0x1a"`
}

type ValidityPeriod struct {
	//+field:string
	NotBefore string `tlv:"0xfe"`
	//+field:string
	NotAfter string `tlv:"0xff"`
}

type CertDescriptionEntry struct {
	//+field:string
	DescriptionKey string `tlv:"0x0201"`
	//+field:string
	DescriptionValue string `tlv:"0x0202"`
}

type CertAdditionalDescription struct {
	//+field:sequence:*CertDescriptionEntry:struct:CertDescriptionEntry
	DescriptionEntries []*CertDescriptionEntry `tlv:"0x0200"`
}

type SignatureInfo struct {
	//+field:natural
	SignatureType uint64 `tlv:"0x1b"`
	//+field:struct:KeyLocator
	KeyLocator *KeyLocator `tlv:"0x1c"`
	//+field:binary
	SignatureNonce []byte `tlv:"0x26"`
	//+field:time:optional
	SignatureTime *time.Duration `tlv:"0x28"`
	//+field:natural:optional
	SignatureSeqNum *uint64 `tlv:"0x2a"`
	//+field:struct:ValidityPeriod
	ValidityPeriod *ValidityPeriod `tlv:"0xfd"`
	//+field:struct:CertAdditionalDescription
	AdditionalDescription *CertAdditionalDescription `tlv:"0x0102"`
}

const (
	NackReasonNone       = uint64(0)
	NackReasonCongestion = uint64(50)
	NackReasonDuplicate  = uint64(100)
	NackReasonNoRoute    = uint64(150)
)

type NetworkNack struct {
	//+field:natural
	Reason uint64 `tlv:"0x0321"`
}

type CachePolicy struct {
	//+field:natural
	CachePolicyType uint64 `tlv:"0x0335"`
}

//+tlv-model:nocopy,private
type LpPacket struct {
	//+field:fixedUint:uint64:optional
	Sequence *uint64 `tlv:"0x51"`
	//+field:natural:optional
	FragIndex *uint64 `tlv:"0x52"`
	//+field:natural:optional
	FragCount *uint64 `tlv:"0x53"`
	//+field:binary
	PitToken []byte `tlv:"0x62"`
	//+field:struct:NetworkNack
	Nack *NetworkNack `tlv:"0x0320"`
	//+field:natural:optional
	IncomingFaceId *uint64 `tlv:"0x032C"`
	//+field:natural:optional
	NextHopFaceId *uint64 `tlv:"0x0330"`
	//+field:struct:CachePolicy
	CachePolicy *CachePolicy `tlv:"0x0334"`
	//+field:natural:optional
	CongestionMark *uint64 `tlv:"0x0340"`
	//+field:fixedUint:uint64:optional
	Ack *uint64 `tlv:"0x0344"`
	//+field:fixedUint:uint64:optional
	TxSequence *uint64 `tlv:"0x0348"`
	//+field:bool
	NonDiscovery bool `tlv:"0x034C"`
	//+field:wire
	PrefixAnnouncement enc.Wire `tlv:"0x0350"`

	//+field:wire
	Fragment enc.Wire `tlv:"0x50"`
}

//+tlv-model:nocopy,private
type Interest struct {
	//+field:procedureArgument:enc.Wire
	sigCovered enc.PlaceHolder
	//+field:procedureArgument:enc.Wire
	digestCovered enc.PlaceHolder

	//+field:interestName:sigCovered
	NameV enc.Name `tlv:"0x07"`
	//+field:bool
	CanBePrefixV bool `tlv:"0x21"`
	//+field:bool
	MustBeFreshV bool `tlv:"0x12"`
	//+field:struct:Links
	ForwardingHintV *Links `tlv:"0x1e"`
	//+field:fixedUint:uint32:optional
	NonceV *uint32 `tlv:"0x0a"`
	//+field:time:optional
	InterestLifetimeV *time.Duration `tlv:"0x0c"`
	//+field:fixedUint:byte:optional
	HopLimitV *byte `tlv:"0x22"`

	//+field:offsetMarker
	sigCoverStart enc.PlaceHolder
	//+field:offsetMarker
	digestCoverStart enc.PlaceHolder

	//+field:wire
	ApplicationParameters enc.Wire `tlv:"0x24"`
	//+field:struct:SignatureInfo
	SignatureInfo *SignatureInfo `tlv:"0x2c"`
	//+field:signature:sigCoverStart:sigCovered
	SignatureValue enc.Wire `tlv:"0x2e"`

	//+field:rangeMarker:digestCoverStart:digestCovered
	digestCoverEnd enc.PlaceHolder
}

//+tlv-model:nocopy,private
type Data struct {
	//+field:procedureArgument:enc.Wire
	sigCovered enc.PlaceHolder
	//+field:offsetMarker
	sigCoverStart enc.PlaceHolder

	//+field:name
	NameV enc.Name `tlv:"0x07"`
	//+field:struct:MetaInfo
	MetaInfo *MetaInfo `tlv:"0x14"`
	//+field:wire
	ContentV enc.Wire `tlv:"0x15"`
	//+field:struct:SignatureInfo
	SignatureInfo *SignatureInfo `tlv:"0x16"`
	//+field:signature:sigCoverStart:sigCovered
	SignatureValue enc.Wire `tlv:"0x17"`
}

//+tlv-model:nocopy,private
type Packet struct {
	//+field:struct:Interest:nocopy
	Interest *Interest `tlv:"0x05"`
	//+field:struct:Data:nocopy
	Data *Data `tlv:"0x06"`
	//+field:struct:LpPacket:nocopy
	LpPacket *LpPacket `tlv:"0x64"`
}

const (
	TypeInterest = enc.TLNum(0x05)
	TypeData     = enc.TLNum(0x06)
	TypeLpPacket = enc.TLNum(0x64)
)
