package codegen

import (
	"fmt"
	"text/template"
)

// NameField represents a name field.
type NameField struct {
	BaseTlvField
}

func NewNameField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &NameField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
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
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_length", f.name), true))
	g.printlnf("l += encoder.%s_length", f.name)
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
	const Temp = `value.{{.Name}} = make(enc.Name, l/2+1)
	startName := reader.Pos()
	endName := startName + int(l)
	for j := range value.{{.Name}} {
		if reader.Pos() >= endName {
			value.{{.Name}} = value.{{.Name}}[:j]
			break
		}
		var err1, err3 error
		value.{{.Name}}[j].Typ, err1 = enc.ReadTLNum(reader)
		l, err2 := enc.ReadTLNum(reader)
		value.{{.Name}}[j].Val, err3 = reader.ReadBuf(int(l))
		if err1 != nil || err2 != nil || err3 != nil {
			err = io.ErrUnexpectedEOF
			break
		}
	}
	if err == nil && reader.Pos() != endName {
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
