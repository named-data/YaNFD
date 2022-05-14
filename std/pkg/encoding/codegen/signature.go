package codegen

import (
	"fmt"
	"text/template"
)

type SignatureField struct {
	BaseTlvField

	sigCovered string
	startPoint string
}

func (f *SignatureField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_signer ndn.Signer", f.name)
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_estLen uint", f.name)
	return g.output()
}

func (f *SignatureField) GenInitEncoder() (string, error) {
	const Temp = `encoder.{{.}}_wireIdx = -1
	if encoder.{{.}}_signer == nil {
		encoder.{{.}}_estLen = 0
	} else {
		encoder.{{.}}_estLen = encoder.{{.}}_signer.EstimateSize()
	}
	`
	var g strErrBuf
	t := template.Must(template.New("SignatureInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

func (f *SignatureField) GenParsingContextStruct() (string, error) {
	return fmt.Sprintf("%s_wireIdx int", f.name), nil
}

func (f *SignatureField) GenInitContext() (string, error) {
	return fmt.Sprintf("context.%s_wireIdx = -1", f.name), nil
}

func (f *SignatureField) GenEncodingLength() (string, error) {
	var g strErrBuf
	g.printlnf("if value.%s_signer != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_estLen", f.name), true))
	g.printlnf("l += encoder.%s_estLen", f.name)
	g.printlnf("}")
	return g.output()
}

func (f *SignatureField) GenEncodingWirePlan() (string, error) {
	// TODO
	panic(nil)
}

type InterestNameField struct {
	BaseTlvField

	// needDigest string
	// digestBuffer string
	sigCovered string
}
