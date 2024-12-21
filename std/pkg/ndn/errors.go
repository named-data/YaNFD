package ndn

import (
	"errors"
	"fmt"
)

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
var ErrFailedToEncode = errors.New("failed to encode an NDN packet")

// ErrWrongType is returned when the type of the packet to parse is not expected.
var ErrWrongType = errors.New("packet to parse is not of desired type")

// ErrPrefixPropViolation is returned when the prefix property is violated during handler registration.
var ErrPrefixPropViolation = errors.New("a prefix or extention of the given handler prefix is already attached")

// ErrDeadlineExceed is returned when the deadline of the Interest passed.
var ErrDeadlineExceed = errors.New("interest deadline exceeded")

// ErrFaceDown is returned when the face is closed.
var ErrFaceDown = errors.New("face is down. Unable to send packet")
