package codegen

import "text/template"

func (f *NaturalField) GenToDict() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlnf("\tdict[\"%s\"] = *value.%s", f.name, f.name)
		g.printlnf("}")
	} else {
		g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	}
	return g.output()
}

func (f *NaturalField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(uint64); ok {")
	if f.opt {
		g.printlnf("\t\tvalue.%s = &v", f.name)
	} else {
		g.printlnf("\t\tvalue.%s = v", f.name)
	}
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"uint64\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	if f.opt {
		g.printlnf("\tvalue.%s = nil", f.name)
	} else {
		g.printlnf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum)
	}
	g.printlnf("}")
	return g.output()
}

func (f *StringField) GenToDict() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlnf("\tdict[\"%s\"] = *value.%s", f.name, f.name)
		g.printlnf("}")
	} else {
		g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	}
	return g.output()
}

func (f *StringField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(string); ok {")
	if f.opt {
		g.printlnf("\t\tvalue.%s = &v", f.name)
	} else {
		g.printlnf("\t\tvalue.%s = v", f.name)
	}
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"string\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	if f.opt {
		g.printlnf("\tvalue.%s = nil", f.name)
	} else {
		g.printlnf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum)
	}
	g.printlnf("}")
	return g.output()
}

func (f *BinaryField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlnf("\tdict[\"%s\"] = value.%s", f.name, f.name)
	g.printlnf("}")
	return g.output()
}

func (f *BinaryField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.([]byte); ok {")
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"[]byte\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = nil", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *BoolField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	return g.output()
}

func (f *BoolField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(bool); ok {")
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"bool\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = false", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *NameField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlnf("\tdict[\"%s\"] = value.%s", f.name, f.name)
	g.printlnf("}")
	return g.output()
}

func (f *NameField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(enc.Name); ok {")
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"Name\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = nil", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *StructField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlnf("\tdict[\"%s\"] = value.%s.ToDict()", f.name, f.name)
	g.printlnf("}")
	return g.output()
}

func (f *StructField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(*%s); ok {", f.StructType)
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"*%s\", Value: vv}",
		f.name, f.typeNum, f.StructType)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = nil", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *SequenceField) GenToDict() (string, error) {
	var g strErrBuf
	// Sequence uses faked encoder variable to embed the subfield.
	// I have verified that the Go compiler can optimize this in simple cases.
	const Temp = `{
		{{.Name}}_l := len(value.{{.Name}})
		dictSeq = make([]{{.FieldType}}, {{.Name}}_l)
		for i := 0; i < {{.Name}}_l; i ++ {
			pseudoValue := struct {
				{{.Name}} {{.FieldType}}
			}{
				{{.Name}}: value.{{.Name}}[i],
			}
			pseudoMap = make(map[string]interface{})
			{
				dict := pseudoMap
				value := &pseudoValue
				{{.SubField.GenToDict}}
				_ = dict
				_ = value
			}
			dictSeq[i] = pseudoMap[{{.Name}}]
		}
		dict[\"{{.Name}}\"] = dictSeq
	}
	`
	t := template.Must(template.New("SeqInitEncoder").Parse(Temp))
	g.executeTemplate(t, f)
	return g.output()
}

func (f *SequenceField) GenFromDict() (string, error) {
	return "ERROR = \"Unimplemented yet!\"", nil
}
