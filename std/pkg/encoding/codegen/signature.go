package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

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

type InterestNameField struct {
	BaseTlvField

	// needDigest string
	// digestBuffer string
	sigCovered string
}
