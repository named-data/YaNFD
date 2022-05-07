package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

var fieldList map[string]func(string, uint64, string, *TlvModel) (TlvField, error)

// NaturalField represents a natural number field.
type NaturalField struct {
	BaseTlvField

	opt bool
}

func (f *NaturalField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("*value."+f.name, false))
		g.printlnf("}")
	} else {
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("value."+f.name, false))
	}
	return g.output()
}

func (f *NaturalField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *NaturalField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("*value."+f.name, false))
		g.printlnf("}")
	} else {
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("value."+f.name, false))
	}
	return g.output()
}

func (f *NaturalField) GenReadFrom() (string, error) {
	if f.opt {
		g := strErrBuf{}
		g.printlnf("{")
		g.printlnf("tempVal := uint64(0)")
		g.printlne(GenNaturalNumberDecode("tempVal"))
		g.printlnf("value." + f.name + " = &tempVal")
		g.printlnf("}")
		return g.output()
	} else {
		return GenNaturalNumberDecode("value." + f.name)
	}
}

func (f *NaturalField) GenSkipProcess() (string, error) {
	if f.opt {
		return "value." + f.name + " = nil", nil
	} else {
		return "err = enc.ErrSkipRequired{TypeNum: typ}", nil
	}
}

func NewNaturalField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &NaturalField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		opt: annotation == "optional",
	}, nil
}

// TimeField represents a time field, recorded as milliseconds.
type TimeField struct {
	BaseTlvField

	opt bool
}

func (f *TimeField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenTypeNumLen(f.TypeNum()))
		g.printlne(GenNaturalNumberLen("uint64(*value."+f.name+"/time.Millisecond)", false))
		g.printlnf("}")
	} else {
		g.printlne(GenTypeNumLen(f.TypeNum()))
		g.printlne(GenNaturalNumberLen("uint64(value."+f.name+"/time.Millisecond)", false))
	}
	return g.output()
}

func (f *TimeField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *TimeField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("uint64(*value."+f.name+"/time.Millisecond)", false))
		g.printlnf("}")
	} else {
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("uint64(value."+f.name+"/time.Millisecond)", false))
	}
	return g.output()
}

func (f *TimeField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("{")
	g.printlnf("timeInt := uint64(0)")
	g.printlne(GenNaturalNumberDecode("timeInt"))
	if f.opt {
		g.printlnf("tempVal := time.Duration(timeInt) * time.Millisecond")
		g.printlnf("value.%s = &tempVal", f.name)
	} else {
		g.printlnf("value.%s = time.Duration(timeInt) * time.Millisecond", f.name)
	}
	g.printlnf("}")
	return g.output()
}

func (f *TimeField) GenSkipProcess() (string, error) {
	if f.opt {
		return "value." + f.name + " = nil", nil
	} else {
		return "err = enc.ErrSkipRequired{TypeNum: typ}", nil
	}
}

func NewTimeField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &TimeField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		opt: annotation == "optional",
	}, nil
}

// BinaryField represents a binary string field of type Buffer or []byte.
// BinaryField always makes a copy during encoding.
type BinaryField struct {
	BaseTlvField
}

func (f *BinaryField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("len(value."+f.name+")", true))
	g.printlnf("l += uint(len(value." + f.name + "))")
	g.printlnf("}")
	return g.output()
}

func (f *BinaryField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *BinaryField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("len(value."+f.name+")", true))
	g.printlnf("copy(buf[pos:], value." + f.name + ")")
	g.printlnf("pos += uint(len(value." + f.name + "))")
	g.printlnf("}")
	return g.output()
}

func (f *BinaryField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s = make([]byte, l)", f.name)
	g.printlnf("_, err = io.ReadFull(reader, value.%s)", f.name)
	return g.output()
}

func (f *BinaryField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

func NewBinaryField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &BinaryField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

// WireField represents a binary string field of type Wire or [][]byte.
type WireField struct {
	BaseTlvField

	noCopy bool
}

func (f *WireField) GenEncoderStruct() (string, error) {
	return fmt.Sprintf("%s_length uint", f.name), nil
}

func (f *WireField) GenInitEncoder() (string, error) {
	var g strErrBuf
	const Temp = `encoder.{{.}}_length = 0
	for _, c := range value.{{.}} {
		encoder.{{.}}_length += uint(len(c))
	}
	`
	t := template.Must(template.New("WireInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.Name)
	return g.output()
}

func (f *WireField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length)", true))
	g.printlnf("l += encoder." + f.name + "_length")
	return g.output()
}

func (f *WireField) GenEncodingWirePlan() (string, error) {
	if f.noCopy {
		g := strErrBuf{}
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("len(value."+f.name+")", true))
		g.printlne(GenSwitchWirePlan())
		g.printlnf("for range value.%s {", f.name)
		g.printlne(GenSwitchWirePlan())
		g.printlnf("}")
		return g.output()
	} else {
		return f.GenEncodingLength()
	}
}

func (f *WireField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("len(value."+f.name+")", true))
	if f.noCopy {
		g.printlne(GenSwitchWire())
		g.printlnf("for _, w := range value.%s {", f.name)
		g.printlnf("wire[wireIdx] = w")
		g.printlne(GenSwitchWire())
		g.printlnf("}")
	} else {
		g.printlnf("for _, w := range value.%s {", f.name)
		g.printlnf("copy(buf[pos:], w)")
		g.printlnf("pos += uint(len(w))")
		g.printlnf("}")
	}
	return g.output()
}

func (f *WireField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s, err = reader.ReadWire(l)", f.name)
	return g.output()
}

func (f *WireField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

func NewWireField(name string, typeNum uint64, _ string, model *TlvModel) (TlvField, error) {
	return &WireField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		noCopy: model.NoCopy,
	}, nil
}

// NameField represents a name field.
type NameField struct {
	BaseTlvField
}

func (f *NameField) GenEncoderStruct() (string, error) {
	return fmt.Sprintf("%s_length uint", f.name), nil
}

func (f *NameField) GenInitEncoder() (string, error) {
	var g strErrBuf
	const Temp = `encoder.{{.}}_length = 0
	for _, c := range value.{{.}} {
		encoder.{{.}}_length += uint(c.EncodingLength())
	}
	`
	t := template.Must(template.New("WireInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.Name)
	return g.output()
}

func (f *NameField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length)", true))
	g.printlnf("l += encoder." + f.name + "_length")
	return g.output()
}

func (f *NameField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *NameField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("len(value."+f.name+")", true))
	g.printlnf("for _, c := range value.%s {", f.name)
	g.printlnf("pos += uint(c.EncodeInto(buf[pos:]))")
	g.printlnf("}")
	return g.output()
}

func (f *NameField) GenReadFrom() (string, error) {
	var g strErrBuf
	const Temp = `value.{{.Name}} = make([][]byte, 0)
	startName := reader.Pos()
	endName := startName + uint(l)
	for reader.Pos() < endName {
		c, err := enc.ReadComponent()
		if err != nil {
			break
		}
		value.{{.Name}} = append(value.{{.Name}}, *c)
	}
	if err != nil && reader.Pos() != endName {
		err = enc.ErrBufferOverflow
	}
	`
	t := template.Must(template.New("NameEncodeInto").Parse(Temp))
	g.executeTemplate(t, f)
	return g.output()
}

func (f *NameField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

func NewNameField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &NameField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

// BoolField represents a boolean field.
type BoolField struct {
	BaseTlvField
}

func (f *BoolField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value." + f.name + " {")
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenTypeNumLen(0))
	g.printlnf("}")
	return g.output()
}

func (f *BoolField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *BoolField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value." + f.name + " {")
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenEncodeTypeNum(0))
	g.printlnf("}")
	return g.output()
}

func (f *BoolField) GenReadFrom() (string, error) {
	return "value." + f.name + " = true", nil
}

func (f *BoolField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = false", nil
}

func NewBoolField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &BoolField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

func CreateField(fieldName string, name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	for k, f := range fieldList {
		if strings.HasPrefix(fieldName, k) {
			return f(name, typeNum, annotation, model)
		}
	}
	return nil, ErrInvalidField
}

func initFields() {
	fieldList = map[string]func(string, uint64, string, *TlvModel) (TlvField, error){
		"natural":           NewNaturalField,
		"time":              NewTimeField,
		"binary":            NewBinaryField,
		"wire":              NewWireField,
		"name":              NewNameField,
		"bool":              NewBoolField,
		"procedureArgument": NewProcedureArgument,
		"offsetMarker":      NewOffsetMarker,
	}
}
