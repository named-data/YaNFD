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

// SigType represents the type of signature.
type SigType int

const (
	SignatureNone            SigType = -1
	SignatureDigestSha256    SigType = 0
	SignatureSha256WithRsa   SigType = 1
	SignatureSha256WithEcdsa SigType = 3
	SignatureHmacWithSha256  SigType = 4
	SignatureEd25519         SigType = 5
)

// ContentType represents the type of Data content in MetaInfo.
type ContentType uint

const (
	ContentTypeBlob ContentType = 0
	ContentTypeLink ContentType = 1
	ContentTypeKey  ContentType = 2
	ContentTypeNack ContentType = 3
)

// InterestResult represents the result of Interest expression.
// Can be Data fetched (succeeded), NetworkNack received, or Timeout.
// Note that AppNack is considered as Data.
type InterestResult int

const (
	InterestResultNone InterestResult = iota
	InterestResultData
	InterestResultNack
	InterestResultTimeout
)

// SigConfig represents the configuration of signature used in signing.
type SigConfig struct {
	Type      SigType
	KeyName   enc.Name
	Nonce     []byte
	SigTime   *time.Time
	SeqNum    *uint64
	NotBefore *time.Time
	NotAfter  *time.Time
}

// Signature is the abstract of the signature of a packet.
// Some of the fields are invalid for Data or Interest.
type Signature interface {
	SigType() SigType
	KeyName() enc.Name
	SigNonce() []byte
	SigTime() *time.Time
	SigSeqNum() *uint64
	Validity() (notBefore, notAfter *time.Time)

	SigValue() []byte
}

// Signer is the interface of the signer of a packet.
type Signer interface {
	SigInfo() (*SigConfig, error)
	EstimateSize() uint
	ComputeSigValue(enc.Wire) ([]byte, error)
}

// DataConfig is used to create a Data.
type DataConfig struct {
	ContentType  *ContentType
	Freshness    *time.Duration
	FinalBlockID *enc.Component
}

// Data is the abstract of a received Data packet.
type Data interface {
	Name() enc.Name
	ContentType() *ContentType
	Freshness() *time.Duration
	FinalBlockID() *enc.Component
	Content() enc.Wire

	Signature() Signature
}

// InterestConfig is used to create a Interest.
type InterestConfig struct {
	CanBePrefix    bool
	MustBeFresh    bool
	ForwardingHint []enc.Name
	Nonce          *uint64
	Lifetime       *time.Duration
	HopLimit       *uint
}

// Interest is the abstract of a received Interest packet.
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
	// MakeData creates a Data packet, returns the encoded Data, signature covered parts, and error.
	MakeData(name enc.Name, config *DataConfig, content enc.Wire, signer Signer) (enc.Wire, enc.Wire, error)
	// MakeData creates an Interest packet, returns the encoded Interest, signature covered parts,
	// the final Interest name, and error.
	MakeInterest(
		name enc.Name, config *InterestConfig, appParam enc.Wire, signer Signer,
	) (enc.Wire, enc.Wire, enc.Name, error)
	// ReadData reads and parses a Data from the reader, returns the Data, signature covered parts, and error.
	ReadData(reader enc.ParseReader) (Data, enc.Wire, error)
	// ReadData reads and parses an Interest from the reader, returns the Data, signature covered parts, and error.
	ReadInterest(reader enc.ParseReader) (Interest, enc.Wire, error)
}

// ReplyFunc represents the callback function to reply for an Interest.
type ReplyFunc func(encodedData enc.Wire) error

// ExpressCallbackFunc represents the callback function for Interest expression.
type ExpressCallbackFunc func(result InterestResult, data Data, rawData enc.Wire, sigCovered enc.Wire) error

// InterestHandler represents the callback function for an Interest handler.
type InterestHandler func(
	interest Interest, rawInterest enc.Wire, sigCovered enc.Wire,
	reply ReplyFunc, deadline time.Time,
)

type Timer interface {
	// Now returns current time.
	Now() time.Time
	// Sleep sleeps for the duration.
	Sleep(time.Duration)
	// Schedule schedules the callback function to be called after the duration,
	// and returns a cancel callback to cancel the scheduled function.
	Schedule(time.Duration, func()) func() error
}

// Engine represents a running NDN App low-level engine.
// Used by NTSchema.
type Engine interface {
	// EngineTrait is the type trait of the NDN engine.
	EngineTrait() Engine
	// Spec returns an NDN packet specification.
	Spec() Spec
	// Timer returns a Timer managed by the engine.
	Timer() Timer
	// AttachHandler attaches an Interest handler to the namespace of prefix.
	// Interest handlers are required to have non-overlapping namespace.
	// That is, one handler's prefix must not be the prefix of another handler's prefix.
	AttachHandler(prefix enc.Name, handler InterestHandler) error
	// DetachHandler detaches an Interest handler from the namespace of prefix.
	DetachHandler(prefix enc.Name) error
	// RegisterRoute registers a route of prefix to the local forwarder.
	RegisterRoute(prefix enc.Name) error
	// UnregisterRoute unregisters a route of prefix to the local forwarder.
	UnregisterRoute(prefix enc.Name) error
	// Express expresses an Interest, with callback called when there is result.
	// To simplify the implementation, finalName needs to be the final Interest name given by MakeInterest.
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
