package codegen

import (
	"bytes"
	"text/template"
)

// TlvModel represents a TLV encodable structure.
type TlvModel struct {
	Name string

	// PrivMethods indicates whether generated methods are private. False by default.
	// Enabled by `private` annotation.
	PrivMethods bool

	// NoCopy indicates whether to avoid copying []byte into wire. False by default.
	// Enabled by `nocopy` annotation.
	NoCopy bool

	// WithParsingContext is true if any field has a non-trivial GenParsingContextStruct()
	WithParsingContext bool

	// Fields are the TLV fields of the structure.
	Fields []TlvField
}

func (m *TlvModel) ProcessOption(option string) {
	switch option {
	case "private":
		m.PrivMethods = true
	case "nocopy":
		m.NoCopy = true
	}
}

func (m *TlvModel) GenEncoderStruct(buf *bytes.Buffer) error {
	const Temp = `type {{.Name}}Encoder struct {
		length uint
		{{if .NoCopy}}
		wirePlan []uint
		{{end}}
		{{- range $f := .Fields}}
		{{$f.GenEncoderStruct}}
		{{- end}}
	}
	`
	t := template.Must(template.New("ModelEncoderStruct").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) GenInitEncoder(buf *bytes.Buffer) error {
	const Temp = `func (encoder *{{.Name}}Encoder) Init(value *{{.Name}}) {
		{{- range $f := .Fields}}
		{{$f.GenInitEncoder}}
		{{- end}}
		l := uint(0)
		{{- range $f := .Fields}}
		{{$f.GenEncodingLength}}
		{{- end}}
		encoder.length = l
		{{if .NoCopy}}
		wirePlan := make([]uint, 0)
		l = uint(0)
		{{- range $f := .Fields}}
		{{$f.GenEncodingWirePlan}}
		{{- end}}
		if l > 0 {
			wirePlan = append(wirePlan, l)
		}
		encoder.wirePlan = wirePlan
		{{- end}}
	}
	`
	t := template.Must(template.New("ModelInitEncoderStruct").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) GenEncodeInto(buf *bytes.Buffer) error {
	const Temp = `func (encoder *{{.Name}}Encoder) EncodeInto(value *{{.Name}},
		{{- if .NoCopy}}wire enc.Wire{{else}}buf []byte{{end}}) {
		{{if .NoCopy}}
		wireIdx := 0
		buf := wire[wireIdx]
		{{end}}
		pos := uint(0)
		{{- range $f := .Fields}}
		{{$f.GenEncodeInto}}
		{{- end}}
	}

	func (encoder *{{.Name}}Encoder) Encode(value *{{.Name}}) enc.Wire {
		{{if .NoCopy}}
		wire := make(enc.Wire, len(encoder.wirePlan))
		for i, l := range encoder.wirePlan {
			if l > 0 {
				wire[i] = make([]byte, l)
			}
		}
		encoder.EncodeInto(value, wire)
		{{else}}
		wire := make(enc.Wire, 1)
		wire[0] = make([]byte, encoder.length)
		buf := wire[0]
		encoder.EncodeInto(value, buf)
		{{end}}
		return wire
	}
	`
	t := template.Must(template.New("ModelEncodeInto").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) GenParsingContextStruct(buf *bytes.Buffer) error {
	const Temp = `type {{.Name}}ParsingContext struct {
		{{- range $f := .Fields}}
		{{$f.GenParsingContextStruct}}
		{{- end}}
	}
	`
	t := template.Must(template.New("ModelParsingContextStruct").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) GenInitContext(buf *bytes.Buffer) error {
	const Temp = `func (context *{{.Name}}ParsingContext) Init() {
		{{- range $f := .Fields}}
		{{$f.GenInitContext}}
		{{- end}}
	}
	`
	t := template.Must(template.New("ModelInitContext").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) GenReadFrom(buf *bytes.Buffer) error {
	const Temp = `
	{{if .Model.WithParsingContext -}}
	func (context *{{.Model.Name}}ParsingContext) Parse
	{{- else -}}
	func {{if .Model.PrivMethods -}}parse{{else}}Parse{{end}}{{.Model.Name}}
	{{- end -}}
	(reader enc.ParseReader, ignoreCritical bool) (*{{.Model.Name}}, error) {
		if reader == nil {
			return nil, enc.ErrBufferOverflow
		}
		progress := -1
		value := &{{.Model.Name}}{}
		var err error
		var startPos int
		for {
			startPos = reader.Pos()
			if startPos >= reader.Length() {
				break
			}
			typ := enc.TLNum(0)
			l := enc.TLNum(0)
			{{call .GenTlvNumberDecode "typ"}}
			{{call .GenTlvNumberDecode "l"}}
			err = nil
			for handled := false; !handled; progress ++ {
				switch typ {
					{{- range $i, $f := .Model.Fields}}
					{{- if (ne $f.TypeNum 0)}}
				case {{$f.TypeNum}}:
					if progress + 1 == {{$i}} {
						handled = true
						{{$f.GenReadFrom}}
					}
					{{- end}}
					{{- end}}
				default:
					handled = true
					if !ignoreCritical && {{.IsCritical}} {
						return nil, enc.ErrUnrecognizedField{TypeNum: typ}
					}
					err = reader.Skip(int(l))
				}
				if err == nil && !handled {
					switch progress {
						{{- range $i, $f := .Model.Fields}}
					case {{$i}} - 1:
						{{$f.GenSkipProcess}}
						{{- end}}
					}
				}
				if err != nil {
					return nil, enc.ErrFailToParse{TypeNum: typ, Err: err}
				}
			}
		}
		startPos = reader.Pos()
		for ; progress < {{len .Model.Fields}}; progress ++ {
			switch progress {
				{{- range $i, $f := .Model.Fields}}
			case {{$i}} - 1:
				{{$f.GenSkipProcess}}
				{{- end}}
			}
		}
		return value, nil
	}
	`
	data := struct {
		Model              *TlvModel
		GenTlvNumberDecode func(string) (string, error)
		IsCritical         string
	}{
		Model:              m,
		GenTlvNumberDecode: GenTlvNumberDecode,
		IsCritical:         `((typ <= 31) || ((typ & 1) == 1))`,
	}
	t := template.Must(template.New("ModelParse").Parse(Temp))
	return t.Execute(buf, data)
}

func (m *TlvModel) detectParsingContext() {
	m.WithParsingContext = false
	for _, f := range m.Fields {
		str, _ := f.GenParsingContextStruct()
		if str != "" {
			m.WithParsingContext = true
		}
	}
}

func (m *TlvModel) genPublicEncode(buf *bytes.Buffer) error {
	const Temp = `func (value *{{.Name}}) Encode() enc.Wire {
		encoder := {{.Name}}Encoder{}
		encoder.Init(value)
		return encoder.Encode(value)
	}

	func (value *{{.Name}}) Bytes() []byte {
		return value.Encode().Join()
	}
	`
	t := template.Must(template.New("PublicEncode").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) genPublicParse(buf *bytes.Buffer) error {
	const Temp = `func Parse{{.Name}}(reader enc.ParseReader, ignoreCritical bool) (*{{.Name}}, error) {
		context := {{.Name}}ParsingContext{}
		context.Init()
		return context.Parse(reader, ignoreCritical)
	}
	`
	t := template.Must(template.New("PublicParse").Parse(Temp))
	return t.Execute(buf, m)
}

func (m *TlvModel) Generate(buf *bytes.Buffer) error {
	// m.detectParsingContext()
	m.WithParsingContext = true
	err := m.GenEncoderStruct(buf)
	if err != nil {
		return err
	}
	buf.WriteRune('\n')
	if m.WithParsingContext {
		err = m.GenParsingContextStruct(buf)
		if err != nil {
			return err
		}
		buf.WriteRune('\n')
	}
	err = m.GenInitEncoder(buf)
	if err != nil {
		return err
	}
	buf.WriteRune('\n')
	if m.WithParsingContext {
		err = m.GenInitContext(buf)
		if err != nil {
			return err
		}
		buf.WriteRune('\n')
	}
	err = m.GenEncodeInto(buf)
	if err != nil {
		return err
	}
	buf.WriteRune('\n')
	err = m.GenReadFrom(buf)
	if err != nil {
		return err
	}
	buf.WriteRune('\n')
	if !m.PrivMethods {
		err = m.genPublicEncode(buf)
		if err != nil {
			return err
		}
		buf.WriteRune('\n')
		if m.WithParsingContext {
			err = m.genPublicParse(buf)
			if err != nil {
				return err
			}
			buf.WriteRune('\n')
		}
	}
	return nil
}
