// Package ndn provides basic interfaces of NDN packet, Specification abstraction, and low-level engine.
// Most high level packages will only depend on ndn, instead of specific implementations.
// Package `ndn.spec_2022` has a default implementation of these interfaces based on current NDN Spec.
package ndn

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type SigType int

const (
	SignatureNone            SigType = -1
	SignatureDigestSha256    SigType = 0
	SignatureSha256WithRsa   SigType = 1
	SignatureSha256WithEcdsa SigType = 3
	SignatureHmacWithSha256  SigType = 4
)

type ContentType uint

const (
	ContentTypeBlob ContentType = 0
	ContentTypeLink ContentType = 1
	ContentTypeKey  ContentType = 2
	ContentTypeNack ContentType = 3
)

type Signature interface {
	SigType() SigType
	SetSigType(SigType)
	KeyName() enc.Name
	SetKeyName(enc.Name)
	Nonce() []byte
	SetNonce([]byte)
	SigTime() *time.Time
	SetSigTime(*time.Time)
	SeqNum() *uint64
	SetSeqNum(*uint64)
	Validity() (notBefore, notAfter *time.Time)
	SetValidity(notBefore, notAfter *time.Time)

	Value() []byte
}

type Data interface {
	Signature() Signature
}

type Interest interface {
	Signature() Signature
}

type Spec interface {
}

type Engine interface {
}

// Interfaces: Data, Interest, Spec, Engine (App lower part)
// High-level API are not required to be zero-copy.
