package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// SignatureField represents SignatureValue field
// It handles the signature covered part, and the position of the signature.
// Requires estimated length of the signature as input, which should be >= the real length.
// When estimated length is 0, the signature is not encoded.
type SignatureField struct {
	BaseTlvField

	sigCovered string
	startPoint string
	noCopy     bool
}

func (f *SignatureField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_estLen uint", f.name)
	return g.output()
}

func (f *SignatureField) GenInitEncoder() (string, error) {
	// SignatureInfo is set in Data/Interest.Encode()
	// {{.}}_estLen is required as an input to the encoder
	const Temp = `encoder.{{.}}_wireIdx = -1
	`
	var g strErrBuf
	t := template.Must(template.New("SignatureInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

func (f *SignatureField) GenParsingContextStruct() (string, error) {
	return "", nil
}

func (f *SignatureField) GenInitContext() (string, error) {
	return fmt.Sprintf("context.%s = make(enc.Wire, 0)", f.sigCovered), nil
}

func (f *SignatureField) GenEncodingLength() (string, error) {
	var g strErrBuf
	g.printlnf("if encoder.%s_estLen > 0 {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_estLen", f.name), true))
	g.printlnf("l += encoder.%s_estLen", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *SignatureField) GenEncodingWirePlan() (string, error) {
	var g strErrBuf
	g.printlnf("if encoder.%s_estLen > 0 {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_estLen", true))
	g.printlne(GenSwitchWirePlan())
	g.printlnf("encoder.%s_wireIdx = len(wirePlan)", f.name)
	g.printlne(GenSwitchWirePlan())
	g.printlnf("}")
	return g.output()
}

func (f *SignatureField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if encoder.%s_estLen > 0 {", f.name)
	g.printlnf("startPos := int(pos)")
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_estLen", true))
	if f.noCopy {
		// Capture the covered part from encoder.startPoint to startPos
		g.printlnf("if encoder.%s_wireIdx == int(wireIdx) {", f.startPoint)
		g.printlnf("coveredPart := buf[encoder.%s:startPos]", f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, coveredPart)", f.sigCovered, f.sigCovered)
		g.printlnf("} else {")
		g.printlnf("coverStart := wire[encoder.%s_wireIdx][encoder.%s:]", f.startPoint, f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, coverStart)", f.sigCovered, f.sigCovered)
		g.printlnf("for i := encoder.%s_wireIdx + 1; i < int(wireIdx); i++ {", f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, wire[i])", f.sigCovered, f.sigCovered)
		g.printlnf("}")
		g.printlnf("coverEnd := buf[:startPos]")
		g.printlnf("encoder.%s = append(encoder.%s, coverEnd)", f.sigCovered, f.sigCovered)
		g.printlnf("}")

		// The outside encoder calculates the signature, so we simply
		// mark the buffer and shuffle the wire.
		g.printlne(GenSwitchWire())
		g.printlne(GenSwitchWire())
	} else {
		g.printlnf("coveredPart := buf[encoder.%s:startPos]", f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, coveredPart)", f.sigCovered, f.sigCovered)

		g.printlnf("pos += encoder.%s_estLen", f.name)
	}
	g.printlnf("}")
	return g.output()
}

func (f *SignatureField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s, err = reader.ReadWire(int(l))", f.name)
	g.printlnf("if err == nil {")
	g.printlnf("coveredPart := reader.Range(context.%s, startPos)", f.startPoint)
	g.printlnf("context.%s = append(context.%s, coveredPart...)", f.sigCovered, f.sigCovered)
	g.printlnf("}")
	return g.output()
}

func (f *SignatureField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

func NewSignatureField(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	strs := strings.Split(annotation, ":")
	if len(strs) < 2 || strs[0] == "" || strs[1] == "" {
		return nil, ErrInvalidField
	}
	return &SignatureField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		startPoint: strs[0],
		sigCovered: strs[1],
		noCopy:     model.NoCopy,
	}, nil
}

// InterestNameField represents the Name field in an Interest, which may contain a ParametersSha256DigestComponent.
// Requires needDigest as input, indicating whether ParametersSha256Digest component is required.
// It will modify the input Name value and generate a final Name value.
type InterestNameField struct {
	BaseTlvField

	sigCovered string
}

func (f *InterestNameField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_length uint", f.name)
	g.printlnf("%s_needDigest bool", f.name)
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_pos uint", f.name)
	return g.output()
}

func (f *InterestNameField) GenInitEncoder() (string, error) {
	var g strErrBuf
	const Temp = `
	encoder.{{.}}_wireIdx = -1
	encoder.{{.}}_length = 0
	if value.{{.}} != nil {
		for _, c := range value.{{.}} {
			encoder.{{.}}_length += uint(c.EncodingLength())
		}
		if encoder.{{.}}_needDigest {
			if len(value.{{.}}) == 0 || value.{{.}}[len(value.{{.}})-1].Typ != enc.TypeParametersSha256DigestComponent {
				encoder.{{.}}_length += 34
			}
		}
	}
	`
	t := template.Must(template.New("InterestNameInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

func (f *InterestNameField) GenParsingContextStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_pos uint", f.name)
	return g.output()
}

func (f *InterestNameField) GenInitContext() (string, error) {
	return "", nil
}

func (f *InterestNameField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
	g.printlnf("l += encoder." + f.name + "_length")
	g.printlnf("}")
	return g.output()
}

func (f *InterestNameField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

func (f *InterestNameField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_length", true))
	g.printlnf("sigCoverStart := pos")

	const Temp = `i := 0
	for i = 0; i < len(value.{{.}}) - 1; i ++ {
		c := value.{{.}}[i]
		pos += uint(c.EncodeInto(buf[pos:]))
	}
	sigCoverEnd := pos
	encoder.{{.}}_wireIdx = int(wireIdx)
	if len(value.{{.}}) > 0 && value.{{.}}[i].Typ == enc.TypeParametersSha256DigestComponent {
		sigCoverEnd = pos
		encoder.{{.}}_pos = pos + 2
		c := value.{{.}}[i]
		pos += uint(c.EncodeInto(buf[pos:]))
	} else {
		if len(value.{{.}}) > 0 {
			c := value.{{.}}[i]
			pos += uint(c.EncodeInto(buf[pos:]))
		}
		sigCoverEnd = pos
		if encoder.{{.}}_needDigest {
			buf[pos] = 0x02
			pos += 1
			buf[pos] = 0x20
			pos += 1
			encoder.{{.}}_pos = pos
			pos += 32
		}
	}
	`
	t := template.Must(template.New("InterestNameEncodeInto").Parse(Temp))
	g.executeTemplate(t, f.name)

	g.printlnf("encoder.%s = append(encoder.%s, buf[sigCoverStart:sigCoverEnd])", f.sigCovered, f.sigCovered)
	g.printlnf("}")
	return g.output()
}

func (f *InterestNameField) GenReadFrom() (string, error) {
	var g strErrBuf

	g.printlnf("{")

	const Temp = `value.{{.Name}} = make(enc.Name, 0)
	startName := reader.Pos()
	endName := startName + int(l)
	sigCoverEnd := endName
	for startComponent := startName; startComponent < endName; startComponent = reader.Pos() {
		c, err := enc.ReadComponent(reader)
		if err != nil {
			break
		}
		value.{{.Name}} = append(value.{{.Name}}, *c)
		if c.Typ == enc.TypeParametersSha256DigestComponent {
			sigCoverEnd = startComponent
		}
	}
	if err == nil && reader.Pos() != endName {
		err = enc.ErrBufferOverflow
	}
	`
	t := template.Must(template.New("NameEncodeInto").Parse(Temp))
	g.executeTemplate(t, f)

	g.printlnf("if err == nil {")
	g.printlnf("coveredPart := reader.Range(startName, sigCoverEnd)")
	g.printlnf("context.%s = append(context.%s, coveredPart...)", f.sigCovered, f.sigCovered)
	g.printlnf("}")

	g.printlnf("}")
	return g.output()
}

func (f *InterestNameField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

func NewInterestNameField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	if annotation == "" {
		return nil, ErrInvalidField
	}
	return &InterestNameField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		sigCovered: annotation,
	}, nil
}
