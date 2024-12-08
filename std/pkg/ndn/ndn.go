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
	SignatureEmptyTest       SigType = 200
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
	// Empty result. Not used by the engine.
	// Used by high-level part if the setting to construct an Interest is incorrect.
	InterestResultNone InterestResult = iota
	// Data is fetched
	InterestResultData
	// NetworkNack is received
	InterestResultNack
	// Timeout
	InterestResultTimeout
	// Cancelled due to disconnection
	InterestCancelled
	// Failed of validation. Not used by the engine itself.
	InterestResultUnverified
	// Other error happens during handling the fetched data. Not used by the engine itself.
	InterestResultError
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
type ExpressCallbackFunc func(result InterestResult, data Data, rawData enc.Wire,
	sigCovered enc.Wire, nackReason uint64)

// InterestHandler represents the callback function for an Interest handler.
// It should create a go routine to avoid blocking the main thread, if either
// 1) Data is not ready to send; or
// 2) Validation is required.
type InterestHandler func(interest Interest, reply ReplyFunc, extra InterestHandlerExtra)

// Extra information passed to the InterestHandler
type InterestHandlerExtra struct {
	// Raw Interest packet wire
	RawInterest enc.Wire
	// Signature covered part of the Interest
	SigCovered enc.Wire
	// Deadline of the Interest
	Deadline time.Time
	// PIT token
	PitToken []byte
	// Incoming face ID (if available)
	IncomingFaceId *uint64
}

// SigChecker is a basic function to check the signature of a packet.
// In NTSchema, policies&sub-trees are supposed to be used for validation;
// SigChecker is only designed for low-level engine.
// Create a go routine for time consuming jobs.
type SigChecker func(name enc.Name, sigCovered enc.Wire, sig Signature) bool

type Timer interface {
	// Now returns current time.
	Now() time.Time
	// Sleep sleeps for the duration.
	Sleep(time.Duration)
	// Schedule schedules the callback function to be called after the duration,
	// and returns a cancel callback to cancel the scheduled function.
	Schedule(time.Duration, func()) func() error
	// Nonce generates a random nonce.
	Nonce() []byte
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
	// The callback should create go routine or channel back to another routine to avoid blocking the main thread.
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

// ErrFailedToEncode is returned when encoding fails but the input arguments are valid.
var ErrFailedToEncode = errors.New("Failed to encode an NDN packet.")

// ErrWrongType is returned when the type of the packet to parse is not expected.
var ErrWrongType = errors.New("Packet to parse is not of desired type.")

// ErrPrefixPropViolation is returned when the prefix property is violated during handler registration.
var ErrPrefixPropViolation = errors.New("A prefix or extention of the given handler prefix is already attached.")

// ErrDeadlineExceed is returned when the deadline of the Interest passed.
var ErrDeadlineExceed = errors.New("Interest deadline exceeded.")

// ErrFaceDown is returned when the face is closed.
var ErrFaceDown = errors.New("Face is down. Unable to send packet.")
