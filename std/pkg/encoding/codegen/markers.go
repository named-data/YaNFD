package codegen

// PlaceHolder is an empty structure that used to give names of procedure arguments.
type PlaceHolder struct{}

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
}

func (f *OffsetMarker) GenEncoderStruct() (string, error) {
	return f.name + " " + "int", nil
}

func (f *OffsetMarker) GenParsingContextStruct() (string, error) {
	return f.name + " " + "int", nil
}

func (f *OffsetMarker) GenReadFrom() (string, error) {
	return f.GenSkipProcess()
}

func (f *OffsetMarker) GenSkipProcess() (string, error) {
	return "context." + f.name + " = reader.Pos()", nil
}

func (f *OffsetMarker) GenEncodeInto() (string, error) {
	return "encoder." + f.name + " = pos", nil
}

// NewOffsetMarker creates an offset marker field.
func NewOffsetMarker(name string, _ uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &OffsetMarker{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: 0,
		},
	}, nil
}
