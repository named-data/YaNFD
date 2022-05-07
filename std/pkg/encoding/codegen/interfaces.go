package codegen

import "errors"

type TlvField interface {
	Name() string
	TypeNum() uint64

	GenEncodingLength() (string, error)
	GenEncodingWirePlan() (string, error)
	GenEncodeInto() (string, error)
	GenEncoderStruct() (string, error)
	GenInitEncoder() (string, error)
	GenParsingContextStruct() (string, error)
	GenInitContext() (string, error)
	GenReadFrom() (string, error)
	GenSkipProcess() (string, error)
}

// BaseTlvField is a base class for all TLV fields.
// Golang's inheritance is not supported, so we use this class to disable
// optional functions.
type BaseTlvField struct {
	name    string
	typeNum uint64
}

func (f *BaseTlvField) Name() string {
	return f.name
}

func (f *BaseTlvField) TypeNum() uint64 {
	return f.typeNum
}

func (*BaseTlvField) GenEncodingLength() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenEncodingWirePlan() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenEncodeInto() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenEncoderStruct() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenInitEncoder() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenParsingContextStruct() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenInitContext() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenReadFrom() (string, error) {
	return "", nil
}

func (*BaseTlvField) GenSkipProcess() (string, error) {
	return "", nil
}

var ErrInvalidField = errors.New("Invalid TlvField. Please check the annotation (including type and arguments)")

var ErrWrongTypeNumber = errors.New("Invalid type number")
