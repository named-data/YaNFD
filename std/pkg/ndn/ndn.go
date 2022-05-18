// Package ndn provides basic interfaces of NDN packet, Specification abstraction, and low-level engine.
// Most high level packages will only depend on ndn, instead of specific implementations.
// To simplify implementation, Data and Interest are immutable.
// Package `ndn.spec_2022` has a default implementation of these interfaces based on current NDN Spec.
package ndn

import (
	"errors"
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

type SigConfig struct {
	Type      SigType
	KeyName   enc.Name
	Nonce     []byte
	SigTime   *time.Time
	SeqNum    *uint64
	NotBefore *time.Time
	NotAfter  *time.Time
}

type Signature interface {
	SigType() SigType
	KeyName() enc.Name
	SigNonce() []byte
	SigTime() *time.Time
	SigSeqNum() *uint64
	Validity() (notBefore, notAfter *time.Time)

	SigValue() []byte
}

type Signer interface {
	SigInfo() (*SigConfig, error)
	EstimateSize() uint
	ComputeSigValue(enc.Wire) ([]byte, error)
}

type DataConfig struct {
	ContentType  *ContentType
	Freshness    *time.Duration
	FinalBlockID *enc.Component
}

type Data interface {
	Name() enc.Name
	ContentType() *ContentType
	Freshness() *time.Duration
	FinalBlockID() *enc.Component
	Content() enc.Wire

	Signature() Signature
}

type InterestConfig struct {
	CanBePrefix    bool
	MustBeFresh    bool
	ForwardingHint []enc.Name
	Nonce          *uint64
	Lifetime       *time.Duration
	HopLimit       *uint
}

type Interest interface {
	Name() enc.Name
	CanBePrefix() bool
	MustBeFresh() bool
	ForwardingHint() []enc.Name
	Nonce() *uint64
	Lifetime() *time.Duration
	HopLimit() *uint
	AppParam() enc.Wire

	Signature() Signature
}

// Spec represents an NDN packet specification.
type Spec interface {
	MakeData(name enc.Name, config *DataConfig, content enc.Wire, signer Signer) (enc.Wire, enc.Wire, error)
	MakeInterest(
		name enc.Name, config *InterestConfig, appParam enc.Wire, signer Signer,
	) (enc.Wire, enc.Wire, enc.Name, error)
	ReadData(reader enc.ParseReader) (Data, enc.Wire, error)
	ReadInterest(reader enc.ParseReader) (Interest, enc.Wire, error)
}

type ReplyFunc func(encodedData enc.Wire) error

type ExpressCallbackFunc func(result InterestResult, data Data, rawData enc.Wire, sigCovered enc.Wire) error

type InterestHandler func(
	interest Interest, rawInterest enc.Wire, sigCovered enc.Wire,
	reply ReplyFunc, deadline time.Time,
)

// Engine represents a running NDN App low-level engine.
// Used by NTSchema.
type Engine interface {
	EngineTrait() Engine
	Spec() Spec
	AttachHandler(prefix enc.Name, handler InterestHandler) error
	DetachHandler(prefix enc.Name) error
	RegisterRoute(prefix enc.Name) error
	UnregisterRoute(prefix enc.Name) error
	Express(finalName enc.Name, config *InterestConfig, rawInterest enc.Wire, callback ExpressCallbackFunc) error
}

type ErrInvalidValue struct {
	Item  string
	Value any
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("Invalid value for %s: %v", e.Item, e.Value)
}

type ErrNotSupported struct {
	Item string
}

func (e ErrNotSupported) Error() string {
	return fmt.Sprintf("Not supported field: %s", e.Item)
}

var ErrFailedToEncode = errors.New("Failed to encode an NDN packet.")

var ErrWrongType = errors.New("Packet to parse is not of desired type.")
