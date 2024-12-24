package ndn

import (
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
)

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

// SigChecker is a basic function to check the signature of a packet.
// In NTSchema, policies&sub-trees are supposed to be used for validation;
// SigChecker is only designed for low-level engine.
// Create a go routine for time consuming jobs.
type SigChecker func(name enc.Name, sigCovered enc.Wire, sig Signature) bool

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
