// Package ndn provides basic interfaces of NDN packet, Specification abstraction, and low-level engine.
// Most high level packages will only depend on ndn, instead of specific implementations.
// Package `ndn.spec_2022` has a default implementation of these interfaces based on current NDN Spec.
package ndn

import (
	"fmt"
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

type InterestResult int

const (
	InterestResultNone InterestResult = iota
	InterestResultData
	InterestResultNack
	InterestResultTimeout
)

type Signature interface {
	SigType() SigType
	SetSigType(SigType) error
	KeyName() enc.Name
	SetKeyName(enc.Name) error
	Nonce() []byte
	SetNonce([]byte) error
	SigTime() *time.Time
	SetSigTime(*time.Time) error
	SeqNum() *uint64
	SetSeqNum(*uint64) error
	Validity() (notBefore, notAfter *time.Time)
	SetValidity(notBefore, notAfter *time.Time) error

	Value() []byte
}

type Signer interface {
	EstimateSize() uint
	SetSigInfo(Signature) error
	ComputeSigValue() ([]byte, error)
}

type Data interface {
	Name() enc.Name
	SetName(enc.Name) error
	ContentType() *ContentType
	SetContentType(*ContentType) error
	Freshness() *time.Duration
	SetFreshness(*time.Duration) error
	FinalBlockID() *enc.Component
	SetFinalBlockID(*enc.Component) error
	Content() enc.Wire
	SetContent(enc.Wire) error

	Signature() Signature
}

type Interest interface {
	Name() enc.Name
	SetName(enc.Name) error
	CanBePrefix() bool
	SetCanBePrefix(bool) error
	MustBeFresh() bool
	SetMustBeFresh(bool) error
	ForwardingHint() []enc.Name
	SetForwardingHint([]enc.Name) error
	Nonce() *uint64
	SetNonce(uint64) error
	Lifetime() *time.Duration
	SetLifetime(*time.Duration) error
	HopLimit() *uint
	SetHopLimit(*uint) error
	AppParam() enc.Wire
	SetAppParam(enc.Wire) error

	Signature() Signature
}

// Spec represents an NDN packet specification.
type Spec interface {
	NewData(name enc.Name, content enc.Wire) Data
	NewInterest(name enc.Name, appParam enc.Wire) Interest
	EncodeData(data Data, signer Signer) (enc.Wire, enc.Wire, error)
	EncodeInterest(interest Interest, signer Signer) (enc.Wire, enc.Wire, error)
	ParseData(wire enc.Wire) (Data, enc.Wire, error)
	ParseInterest(wire enc.Wire) (Interest, enc.Wire, error)
}

type ReplyFunc func(encodedData enc.Wire) error

type ExpressCallbackFunc func(result InterestResult, data Data, rawData enc.Wire, sigCovered enc.Wire) error

type InterestHandler func(interest Interest, rawInterest enc.Wire, sigCovered enc.Wire, reply ReplyFunc)

// Engine represents a running NDN App low-level engine.
// Used by NTSchema.
type Engine interface {
	EngineTrait() Engine
	Spec() Spec
	AttachHandler(prefix enc.Name, handler InterestHandler) error
	DetachHandler(prefix enc.Name) error
	RegisterRoute(prefix enc.Name) error
	UnregisterRoute(prefix enc.Name) error
	Express(finalName enc.Name, rawInterest enc.Wire, callback ExpressCallbackFunc) error
}

type ErrInvalidValue struct {
	Item  string
	Value any
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("Invalid value for %s: %v", e.Item, e.Value)
}
