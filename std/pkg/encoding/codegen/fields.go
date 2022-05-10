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
		return fmt.Sprintf("err = enc.ErrSkipRequired{TypeNum: %d}", f.typeNum), nil
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
		return fmt.Sprintf("err = enc.ErrSkipRequired{TypeNum: %d}", f.typeNum), nil
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
	const Temp = `if value.{{.}} != nil {
		encoder.{{.}}_length = 0
		for _, c := range value.{{.}} {
			encoder.{{.}}_length += uint(len(c))
		}
	}
	`
	t := template.Must(template.New("WireInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

func (f *WireField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
	g.printlnf("l += encoder." + f.name + "_length")
	g.printlnf("}")
	return g.output()
}

func (f *WireField) GenEncodingWirePlan() (string, error) {
	if f.noCopy {
		g := strErrBuf{}
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
		g.printlne(GenSwitchWirePlan())
		g.printlnf("for range value.%s {", f.name)
		g.printlne(GenSwitchWirePlan())
		g.printlnf("}")
		g.printlnf("}")
		return g.output()
	} else {
		return f.GenEncodingLength()
	}
}

func (f *WireField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_length", true))
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
	g.printlnf("}")
	return g.output()
}

func (f *WireField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s, err = reader.ReadWire(int(l))", f.name)
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
	const Temp = `if value.{{.}} != nil {
		encoder.{{.}}_length = 0
		for _, c := range value.{{.}} {
			encoder.{{.}}_length += uint(c.EncodingLength())
		}
	}
	`
	t := template.Must(template.New("NameInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

func (f *NameField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
	g.printlnf("l += encoder." + f.name + "_length")
	g.printlnf("}")
	return g.output()
}

func (f *NameField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *NameField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_length", true))
	g.printlnf("for _, c := range value.%s {", f.name)
	g.printlnf("pos += uint(c.EncodeInto(buf[pos:]))")
	g.printlnf("}")
	g.printlnf("}")
	return g.output()
}

func (f *NameField) GenReadFrom() (string, error) {
	var g strErrBuf
	const Temp = `value.{{.Name}} = make(enc.Name, 0)
	startName := reader.Pos()
	endName := startName + int(l)
	for reader.Pos() < endName {
		c, err := enc.ReadComponent(reader)
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

// SequenceField represents a slice field of another supported field type.
type SequenceField struct {
	BaseTlvField

	SubField  TlvField
	FieldType string
}

func (f *SequenceField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_subencoder []struct{", f.name)
	g.printlne(f.SubField.GenEncoderStruct())
	g.printlnf("}")
	return g.output()
}

func (f *SequenceField) GenInitEncoder() (string, error) {
	var g strErrBuf
	// Sequence uses faked encoder variable to embed the subfield.
	// I have verified that the Go compiler can optimize this in simple cases.
	const Temp = `{
		{{.Name}}_l := len(value.{{.Name}})
		encoder.{{.Name}}_subencoder = make([]struct{
			{{.SubField.GenEncoderStruct}}
		}, {{.Name}}_l)
		for i := 0; i < {{.Name}}_l; i ++ {
			pseudoEncoder := &encoder.{{.Name}}_subencoder[i]
			pseudoValue := struct {
				{{.Name}} {{.FieldType}}
			}{
				{{.Name}}: value.{{.Name}}[i],
			}
			{
				encoder := pseudoEncoder
				value := &pseudoValue
				{{.SubField.GenInitEncoder}}
				_ = encoder
				_ = value
			}
		}
	}
	`
	t := template.Must(template.New("SeqInitEncoder").Parse(Temp))
	g.executeTemplate(t, f)
	return g.output()
}

func (f *SequenceField) GenParsingContextStruct() (string, error) {
	// This is not a slice, because the number of elements is unknown before parsing.
	return f.SubField.GenParsingContextStruct()
}

func (f *SequenceField) GenInitContext() (string, error) {
	return f.SubField.GenInitContext()
}

func (f *SequenceField) encodingGeneral(funcName string) (string, error) {
	var g strErrBuf
	const TempFmt = `if value.{{.Name}} != nil {
			for seq_i, seq_v := range value.{{.Name}} {
			pseudoEncoder := &encoder.{{.Name}}_subencoder[seq_i]
			pseudoValue := struct {
				{{.Name}} {{.FieldType}}
			}{
				{{.Name}}: seq_v,
			}
			{
				encoder := pseudoEncoder
				value := &pseudoValue
				{{.SubField.%s}}
				_ = encoder
				_ = value
			}
		}
	}
	`
	temp := fmt.Sprintf(TempFmt, funcName)
	t := template.Must(template.New("SequenceEncodingGeneral").Parse(temp))
	g.executeTemplate(t, f)
	return g.output()
}

func (f *SequenceField) GenEncodingLength() (string, error) {
	return f.encodingGeneral("GenEncodingLength")
}

func (f *SequenceField) GenEncodingWirePlan() (string, error) {
	return f.encodingGeneral("GenEncodingWirePlan")
}

func (f *SequenceField) GenEncodeInto() (string, error) {
	return f.encodingGeneral("GenEncodeInto")
}

func (f *SequenceField) GenReadFrom() (string, error) {
	var g strErrBuf
	const Temp = `if value.{{.Name}} == nil {
		value.{{.Name}} = make([]{{.FieldType}}, 0)
	}
	{
		pseudoValue := struct {
			{{.Name}} {{.FieldType}}
		}{}
		{
			value := &pseudoValue
			{{.SubField.GenReadFrom}}
			_ = value
		}
		value.{{.Name}} = append(value.{{.Name}}, pseudoValue.{{.Name}})
	}
	progress --
	`
	t := template.Must(template.New("NameEncodeInto").Parse(Temp))
	g.executeTemplate(t, f)
	return g.output()
}

func (f *SequenceField) GenSkipProcess() (string, error) {
	// Skip is called after all elements are parsed, so we should not assign nil.
	return "", nil
}

func NewSequenceField(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	strs := strings.SplitN(annotation, ":", 3)
	if len(strs) < 2 {
		return nil, ErrInvalidField
	}
	subFieldType := strs[0]
	subFieldClass := strs[1]
	if len(strs) >= 3 {
		annotation = strs[2]
	} else {
		annotation = ""
	}
	subField, err := CreateField(subFieldClass, name, typeNum, annotation, model)
	if err != nil {
		return nil, err
	}
	return &SequenceField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		SubField:  subField,
		FieldType: subFieldType,
	}, nil
}

// StructField represents a struct field of another TlvModel.
type StructField struct {
	BaseTlvField

	StructType  string
	innerNoCopy bool
}

func (f *StructField) GenEncoderStruct() (string, error) {
	return fmt.Sprintf("%s_encoder %sEncoder", f.name, f.StructType), nil
}

func (f *StructField) GenInitEncoder() (string, error) {
	const Temp = `if value.{{.}} != nil {
		encoder.{{.}}_encoder.Init(value.{{.}})
	}`
	var g strErrBuf
	t := template.Must(template.New("StructInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

func (f *StructField) GenParsingContextStruct() (string, error) {
	return fmt.Sprintf("%s_context %sParsingContext", f.name, f.StructType), nil
}

func (f *StructField) GenInitContext() (string, error) {
	return fmt.Sprintf("context.%s_context.Init()", f.name), nil
}

func (f *StructField) GenEncodingLength() (string, error) {
	var g strErrBuf
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_encoder.length", f.name), true))
	g.printlnf("l += encoder.%s_encoder.length", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *StructField) GenEncodingWirePlan() (string, error) {
	if f.innerNoCopy {
		var g strErrBuf
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_encoder.length", f.name), true))
		g.printlnf("if encoder.%s_encoder.length > 0 {", f.name)
		// wirePlan[0] is always nonzero.
		g.printlnf("l += encoder.%s_encoder.wirePlan[0]", f.name)
		g.printlnf("for i := 1; i < len(encoder.%s_encoder.wirePlan); i ++ {", f.name)
		g.printlne(GenSwitchWirePlan())
		g.printlnf("l = encoder.%s_encoder.wirePlan[i]", f.name)
		g.printlnf("}")
		// If l == 0 then inner struct ends with a Wire. So we cannot continue.
		// Otherwise, continue on the last part of the inner wire.
		// Therefore, if the inner structure only uses 1 buf (i.e. with no Wire field),
		// the outer structure will not create extra buffers.
		g.printlnf("if l == 0 {")
		g.printlne(GenSwitchWirePlan())
		g.printlnf("}")
		g.printlnf("}")
		g.printlnf("}")
		return g.output()
	} else {
		return f.GenEncodingLength()
	}
}

func (f *StructField) GenEncodeInto() (string, error) {
	var g strErrBuf
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode(fmt.Sprintf("encoder.%s_encoder.length", f.name), true))
	g.printlnf("if encoder.%s_encoder.length > 0 {", f.name)
	if !f.innerNoCopy {
		g.printlnf("encoder.%s_encoder.EncodeInto(value.%s, buf[pos:])", f.name, f.name)
		g.printlnf("pos += encoder.%s_encoder.length", f.name)
	} else {
		const Temp = `{
			subWire := make(enc.Wire, len(encoder.{{.}}_encoder.wirePlan))
			subWire[0] = buf[pos:]
			for i := 1; i < len(subWire); i ++ {
				subWire[i] = wire[wireIdx + i]
			}
			encoder.{{.}}_encoder.EncodeInto(value.{{.}}, subWire)
			for i := 1; i < len(subWire); i ++ {
				wire[wireIdx + i] = subWire[i]
			}
			if lastL := encoder.{{.}}_encoder.wirePlan[len(subWire)-1]; lastL > 0 {
				wireIdx += len(subWire) - 1
				if len(subWire) > 1 {
					pos = lastL
				} else {
					pos += lastL
				}
			} else {
				wireIdx += len(subWire)
				pos = 0
			}
			if wireIdx < len(wire) {
				buf = wire[wireIdx]
			} else {
				buf = nil
			}
		}
		`
		t := template.Must(template.New("StructEncodeInto").Parse(Temp))
		g.executeTemplate(t, f.name)
	}
	g.printlnf("}")
	g.printlnf("}")
	return g.output()
}

func (f *StructField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

func (f *StructField) GenReadFrom() (string, error) {
	ret := fmt.Sprintf("value.%s, err = context.%s_context.Parse(reader.Delegate(int(l)), ignoreCritical)",
		f.name, f.name)
	return ret, nil
}

func NewStructField(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	if annotation == "" {
		return nil, ErrInvalidField
	}
	strs := strings.Split(annotation, ":")
	structType := strs[0]
	innerNoCopy := false
	if len(strs) > 1 && strs[1] == "nocopy" {
		innerNoCopy = true
	}
	if !model.NoCopy && innerNoCopy {
		return nil, ErrInvalidField
	}
	return &StructField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		StructType:  structType,
		innerNoCopy: innerNoCopy,
	}, nil
}
