package codegen

import "fmt"

// NaturalField represents a natural number field.
type NaturalField struct {
	BaseTlvField

	opt bool
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
		return fmt.Sprintf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum), nil
	}
}
