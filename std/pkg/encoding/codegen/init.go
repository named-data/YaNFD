package codegen

import "strings"

var fieldList map[string]func(string, uint64, string, *TlvModel) (TlvField, error)

func init() {
	initFields()
}

func initFields() {
	fieldList = map[string]func(string, uint64, string, *TlvModel) (TlvField, error){
		"natural":           NewNaturalField,
		"fixedUint":         NewFixedUintField,
		"time":              NewTimeField,
		"binary":            NewBinaryField,
		"string":            NewStringField,
		"wire":              NewWireField,
		"name":              NewNameField,
		"bool":              NewBoolField,
		"procedureArgument": NewProcedureArgument,
		"offsetMarker":      NewOffsetMarker,
		"rangeMarker":       NewRangeMarker,
		"sequence":          NewSequenceField,
		"struct":            NewStructField,
		"signature":         NewSignatureField,
		"interestName":      NewInterestNameField,
		"map":               NewMapField,
	}
}

func CreateField(className string, name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	for k, f := range fieldList {
		if strings.HasPrefix(className, k) {
			return f(name, typeNum, annotation, model)
		}
	}
	return nil, ErrInvalidField
}
