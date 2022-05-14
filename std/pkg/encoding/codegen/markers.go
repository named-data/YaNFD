package codegen

// ProcedureArgument is a variable used during encoding and decoding procedure.
type ProcedureArgument struct {
	BaseTlvField

	argType string
}

func (f *ProcedureArgument) GenEncoderStruct() (string, error) {
	return f.name + " " + f.argType, nil
}

func (f *ProcedureArgument) GenParsingContextStruct() (string, error) {
	return f.name + " " + f.argType, nil
}

// NewProcedureArgument creates a ProcedureArgument field.
func NewProcedureArgument(name string, _ uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &ProcedureArgument{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: 0,
		},
		argType: annotation,
	}, nil
}

// OffsetMarker is a marker that marks a position in the wire.
type OffsetMarker struct {
	BaseTlvField

	noCopy bool
}

func (f *OffsetMarker) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s int", f.name)
	if f.noCopy {
		g.printlnf("%s_wireIdx int", f.name)
	}
	g.printlnf("%s_pos int", f.name)
	return g.output()
}

func (f *OffsetMarker) GenParsingContextStruct() (string, error) {
	return f.name + " " + "int", nil
}

func (f *OffsetMarker) GenReadFrom() (string, error) {
	return f.GenSkipProcess()
}

func (f *OffsetMarker) GenSkipProcess() (string, error) {
	return "context." + f.name + " = int(startPos)", nil
}

func (f *OffsetMarker) GenEncodingLength() (string, error) {
	return "encoder." + f.name + " = int(l)", nil
}

func (f *OffsetMarker) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.noCopy {
		g.printlnf("encoder.%s_wireIdx = int(wireIdx)", f.name)
	}
	g.printlnf("encoder.%s_pos = int(pos)", f.name)
	return g.output()
}

// NewOffsetMarker creates an offset marker field.
func NewOffsetMarker(name string, _ uint64, _ string, model *TlvModel) (TlvField, error) {
	return &OffsetMarker{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: 0,
		},
		noCopy: model.NoCopy,
	}, nil
}
