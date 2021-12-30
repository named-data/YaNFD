/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/named-data/YaNFD/ndn/tlv"
	"github.com/named-data/YaNFD/ndn/util"
)

// NameComponent represents an NDN name component.
type NameComponent interface {
	String() string
	DeepCopy() NameComponent
	Type() uint16
	Value() []byte
	Equals(other NameComponent) bool
	Encode() *tlv.Block
}

// DecodeNameComponent decodes a name component from the wire.
func DecodeNameComponent(wire *tlv.Block) (NameComponent, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}

	var n NameComponent
	var err error
	switch wire.Type() {
	case tlv.ImplicitSha256DigestComponent:
		n = NewImplicitSha256DigestComponent(wire.Value())
	case tlv.ParametersSha256DigestComponent:
		n = NewParametersSha256DigestComponent(wire.Value())
	case tlv.GenericNameComponent:
		n = NewGenericNameComponent(wire.Value())
	case tlv.KeywordNameComponent:
		n = NewKeywordNameComponent(wire.Value())
	case tlv.SegmentNameComponent:
		n = DecodeSegmentNameComponent(wire.Value())
	case tlv.ByteOffsetNameComponent:
		n = DecodeByteOffsetNameComponent(wire.Value())
	case tlv.VersionNameComponent:
		n = DecodeVersionNameComponent(wire.Value())
	case tlv.TimestampNameComponent:
		n = DecodeTimestampNameComponent(wire.Value())
	case tlv.SequenceNumNameComponent:
		n = DecodeSequenceNumNameComponent(wire.Value())
	default:
		if wire.Type() > math.MaxUint16 {
			n = nil
			err = util.ErrOutOfRange
		} else {
			n = NewBaseNameComponent(uint16(wire.Type()), wire.Value())
		}
	}

	if n == nil {
		err = util.ErrDecodeNameComponent
	}
	return n, err
}

////////////////////
// BaseNameComponent
////////////////////

// BaseNameComponent represents a name component without a specialized type.
type BaseNameComponent struct {
	tlvType uint16
	value   []byte
	wire    *tlv.Block
}

// NewBaseNameComponent creates a name component of an arbitrary type.
func NewBaseNameComponent(tlvType uint16, value []byte) *BaseNameComponent {
	n := new(BaseNameComponent)
	n.tlvType = tlvType
	n.value = value
	return n
}

func (n *BaseNameComponent) String() string {
	return strconv.FormatUint(uint64(n.tlvType), 10) + "=" + escapeComponent(n.value)
}

// DeepCopy makes a deep copy of the name component.
func (n *BaseNameComponent) DeepCopy() NameComponent {
	newN := new(BaseNameComponent)
	newN.tlvType = n.tlvType
	newN.value = make([]byte, len(n.value))
	copy(newN.value, n.value)
	return newN
}

// Type returns the TLV type of the name component.
func (n *BaseNameComponent) Type() uint16 {
	return n.tlvType
}

// Value returns the TLV value of the name component.
func (n *BaseNameComponent) Value() []byte {
	return n.value
}

// Equals returns whether the two name components match.
func (n *BaseNameComponent) Equals(other NameComponent) bool {
	return n.tlvType == other.Type() && bytes.Equal(n.value, other.Value())
}

// Encode encodes the name component into a block.
func (n *BaseNameComponent) Encode() *tlv.Block {
	if n.wire == nil {
		n.wire = tlv.NewBlock(uint32(n.tlvType), n.value)
		n.wire.Wire()
	}
	return n.wire
}

////////////////////////////////
// ImplicitSha256DigestComponent
////////////////////////////////

// ImplicitSha256DigestComponent represents an implicit SHA-256 digest component.
type ImplicitSha256DigestComponent struct {
	BaseNameComponent
}

// NewImplicitSha256DigestComponent creates a new ImplicitSha256DigestComponent.
func NewImplicitSha256DigestComponent(value []byte) *ImplicitSha256DigestComponent {
	if len(value) != 32 {
		return nil
	}

	n := new(ImplicitSha256DigestComponent)
	n.tlvType = tlv.ImplicitSha256DigestComponent
	n.value = value
	return n
}

func (n *ImplicitSha256DigestComponent) String() string {
	return "sha256digest=" + hex.EncodeToString(n.value)
}

// DeepCopy creates a deep copy of the name component.
func (n *ImplicitSha256DigestComponent) DeepCopy() NameComponent {
	return &ImplicitSha256DigestComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent)}
}

// SetValue sets the value of an ImplicitSha256DigestComponent.
func (n *ImplicitSha256DigestComponent) SetValue(value []byte) error {
	if len(value) != 32 {
		return util.ErrOutOfRange
	}
	n.value = value
	n.wire = nil
	return nil
}

//////////////////////////////////
// ParametersSha256DigestComponent
//////////////////////////////////

// ParametersSha256DigestComponent represents a component containing the SHA-256 digest of the Interest parameters.
type ParametersSha256DigestComponent struct {
	BaseNameComponent
}

// NewParametersSha256DigestComponent creates a new ParametersSha256DigestComponent.
func NewParametersSha256DigestComponent(value []byte) *ParametersSha256DigestComponent {
	if len(value) != 32 {
		return nil
	}

	n := new(ParametersSha256DigestComponent)
	n.tlvType = tlv.ParametersSha256DigestComponent
	n.value = value
	return n
}

func (n *ParametersSha256DigestComponent) String() string {
	return "params-sha256=" + hex.EncodeToString(n.value)
}

// DeepCopy creates a deep copy of the name component.
func (n *ParametersSha256DigestComponent) DeepCopy() NameComponent {
	return &ParametersSha256DigestComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent)}
}

// SetValue sets the value of an ParametersSha256DigestComponent.
func (n *ParametersSha256DigestComponent) SetValue(value []byte) error {
	if len(value) != 32 {
		return util.ErrOutOfRange
	}
	n.value = value
	n.wire = nil
	return nil
}

///////////////////////
// GenericNameComponent
///////////////////////

// GenericNameComponent represents a generic NDN name component.
type GenericNameComponent struct {
	BaseNameComponent
}

// NewGenericNameComponent creates a new GenericNameComponent.
func NewGenericNameComponent(value []byte) *GenericNameComponent {
	n := new(GenericNameComponent)
	n.tlvType = tlv.GenericNameComponent
	n.value = value
	return n
}

func (n *GenericNameComponent) String() string {
	return escapeComponent(n.value)
}

// DeepCopy creates a deep copy of the name component.
func (n *GenericNameComponent) DeepCopy() NameComponent {
	return &GenericNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent)}
}

// SetValue sets the value of a GenericNameComponent.
func (n *GenericNameComponent) SetValue(value []byte) {
	n.value = value
	n.wire = nil
}

///////////////////////
// KeywordNameComponent
///////////////////////

// KeywordNameComponent is a component containing a well-known keyword.
type KeywordNameComponent struct {
	BaseNameComponent
}

// NewKeywordNameComponent creates a new KeywordNameComponent.
func NewKeywordNameComponent(value []byte) *KeywordNameComponent {
	n := new(KeywordNameComponent)
	n.tlvType = tlv.KeywordNameComponent
	n.value = value
	return n
}

func (n *KeywordNameComponent) String() string {
	return escapeComponent(n.value)
}

// DeepCopy creates a deep copy of the name component.
func (n *KeywordNameComponent) DeepCopy() NameComponent {
	return &KeywordNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent)}
}

// SetValue sets the value of a KeywordNameComponent.
func (n *KeywordNameComponent) SetValue(value []byte) {
	n.value = value
	n.wire = nil
}

///////////////////////
// SegmentNameComponent
///////////////////////

// SegmentNameComponent is a component containing a segment number.
type SegmentNameComponent struct {
	BaseNameComponent
	rawValue uint64
}

// NewSegmentNameComponent creates a new SegmentNameComponent.
func NewSegmentNameComponent(value uint64) *SegmentNameComponent {
	n := new(SegmentNameComponent)
	n.tlvType = tlv.SegmentNameComponent
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	return n
}

// DecodeSegmentNameComponent decodes a SegmentNameComponent from a TLV wire value.
func DecodeSegmentNameComponent(value []byte) *SegmentNameComponent {
	n := new(SegmentNameComponent)
	n.tlvType = tlv.SegmentNameComponent
	n.value = value
	var err error
	n.rawValue, err = tlv.DecodeNNI(n.value)
	if err != nil {
		return nil
	}
	return n
}

func (n *SegmentNameComponent) String() string {
	return "seg=" + strconv.FormatUint(n.rawValue, 10)
}

// DeepCopy creates a deep copy of the name component.
func (n *SegmentNameComponent) DeepCopy() NameComponent {
	return &SegmentNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent), rawValue: n.rawValue}
}

// SetValue sets the value of a KeywordNameComponent.
func (n *SegmentNameComponent) SetValue(value uint64) {
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	n.wire = nil
}

//////////////////////////
// ByteOffsetNameComponent
//////////////////////////

// ByteOffsetNameComponent is a component containing a byte offset.
type ByteOffsetNameComponent struct {
	BaseNameComponent
	rawValue uint64
}

// NewByteOffsetNameComponent creates a new ByteOffsetNameComponent.
func NewByteOffsetNameComponent(value uint64) *ByteOffsetNameComponent {
	n := new(ByteOffsetNameComponent)
	n.tlvType = tlv.ByteOffsetNameComponent
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	return n
}

// DecodeByteOffsetNameComponent decodes a ByteOffsetNameComponent from a TLV wire value.
func DecodeByteOffsetNameComponent(value []byte) *ByteOffsetNameComponent {
	n := new(ByteOffsetNameComponent)
	n.tlvType = tlv.ByteOffsetNameComponent
	n.value = value
	var err error
	n.rawValue, err = tlv.DecodeNNI(n.value)
	if err != nil {
		return nil
	}
	return n
}

func (n *ByteOffsetNameComponent) String() string {
	return "off=" + strconv.FormatUint(n.rawValue, 10)
}

// DeepCopy creates a deep copy of the name component.
func (n *ByteOffsetNameComponent) DeepCopy() NameComponent {
	return &ByteOffsetNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent), rawValue: n.rawValue}
}

// SetValue sets the value of a ByteOffsetNameComponent.
func (n *ByteOffsetNameComponent) SetValue(value uint64) {
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	n.wire = nil
}

///////////////////////
// VersionNameComponent
///////////////////////

// VersionNameComponent is a component containing a version number.
type VersionNameComponent struct {
	BaseNameComponent
	rawValue uint64
}

// NewVersionNameComponent creates a new VersionNameComponent.
func NewVersionNameComponent(value uint64) *VersionNameComponent {
	n := new(VersionNameComponent)
	n.tlvType = tlv.VersionNameComponent
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	return n
}

// DecodeVersionNameComponent decodes a VersionNameComponent from a TLV wire value.
func DecodeVersionNameComponent(value []byte) *VersionNameComponent {
	n := new(VersionNameComponent)
	n.tlvType = tlv.VersionNameComponent
	n.value = value
	var err error
	n.rawValue, err = tlv.DecodeNNI(n.value)
	if err != nil {
		return nil
	}
	return n
}

func (n *VersionNameComponent) String() string {
	return "v=" + strconv.FormatUint(n.rawValue, 10)
}

// DeepCopy creates a deep copy of the name component.
func (n *VersionNameComponent) DeepCopy() NameComponent {
	return &VersionNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent), rawValue: n.rawValue}
}

// SetValue sets the value of a VersionNameComponent.
func (n *VersionNameComponent) SetValue(value uint64) {
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	n.wire = nil
}

// Version returns the version contained in the name component.
func (n *VersionNameComponent) Version() uint64 {
	return n.rawValue
}

/////////////////////////
// TimestampNameComponent
/////////////////////////

// TimestampNameComponent is a component containing a Unix timestamp (in microseconds).
type TimestampNameComponent struct {
	BaseNameComponent
	rawValue uint64
}

// NewTimestampNameComponent creates a new TimestampNameComponent.
func NewTimestampNameComponent(value uint64) *TimestampNameComponent {
	n := new(TimestampNameComponent)
	n.tlvType = tlv.TimestampNameComponent
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	return n
}

// DecodeTimestampNameComponent decodes a TimestampNameComponent from a TLV wire value.
func DecodeTimestampNameComponent(value []byte) *TimestampNameComponent {
	n := new(TimestampNameComponent)
	n.tlvType = tlv.TimestampNameComponent
	n.value = value
	var err error
	n.rawValue, err = tlv.DecodeNNI(n.value)
	if err != nil {
		return nil
	}
	return n
}

func (n *TimestampNameComponent) String() string {
	return "t=" + strconv.FormatUint(n.rawValue, 10)
}

// DeepCopy creates a deep copy of the name component.
func (n *TimestampNameComponent) DeepCopy() NameComponent {
	return &TimestampNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent), rawValue: n.rawValue}
}

// SetValue sets the value of a TimestampNameComponent.
func (n *TimestampNameComponent) SetValue(value uint64) {
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	n.wire = nil
}

///////////////////////////
// SequenceNumNameComponent
///////////////////////////

// SequenceNumNameComponent is a component containing a sequence number.
type SequenceNumNameComponent struct {
	BaseNameComponent
	rawValue uint64
}

// NewSequenceNumNameComponent creates a new SequenceNumNameComponent.
func NewSequenceNumNameComponent(value uint64) *SequenceNumNameComponent {
	n := new(SequenceNumNameComponent)
	n.tlvType = tlv.SequenceNumNameComponent
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	return n
}

// DecodeSequenceNumNameComponent decodes a SequenceNumNameComponent from a TLV wire value.
func DecodeSequenceNumNameComponent(value []byte) *SequenceNumNameComponent {
	n := new(SequenceNumNameComponent)
	n.tlvType = tlv.SequenceNumNameComponent
	n.value = value
	var err error
	n.rawValue, err = tlv.DecodeNNI(n.value)
	if err != nil {
		return nil
	}
	return n
}

func (n *SequenceNumNameComponent) String() string {
	return "seq=" + strconv.FormatUint(n.rawValue, 10)
}

// DeepCopy creates a deep copy of the name component.
func (n *SequenceNumNameComponent) DeepCopy() NameComponent {
	return &SequenceNumNameComponent{BaseNameComponent: *n.BaseNameComponent.DeepCopy().(*BaseNameComponent), rawValue: n.rawValue}
}

// SetValue sets the value of a SequenceNumNameComponent.
func (n *SequenceNumNameComponent) SetValue(value uint64) {
	n.rawValue = value
	n.value = tlv.EncodeNNI(n.rawValue)
	n.wire = nil
}

///////
// Name
///////

// Name represents an NDN name.
type Name struct {
	components   []NameComponent
	wire         *tlv.Block
	cachedString string
}

// NewName constructs an empty name.
func NewName() *Name {
	n := new(Name)
	return n
}

// NameFromString decodes a name from a string.
func NameFromString(str string) (*Name, error) {
	n := new(Name)

	if len(str) == 0 {
		// Empty name
		return n, nil
	}

	components := strings.Split(str, "/")[1:] // Skip first since empty
	if len(components[0]) == 0 {
		// Empty name
		return n, nil
	}
	for _, component := range components {
		var c NameComponent
		if strings.Contains(component, "=") {
			componentSplit := strings.Split(component, "=")
			if len(componentSplit) != 2 {
				return nil, errors.New("Name component has extraneous =")
			}

			unescapedValue, err := unescapeComponent(componentSplit[1])
			if err != nil {
				return nil, errors.New("error unescaping component value")
			}

			switch componentSplit[0] {
			case "sha256digest":
				digest, err := hex.DecodeString(unescapedValue)
				if err != nil {
					return nil, errors.New("ImplicitSha256DigestComponent is not a hex string")
				}
				c = NewImplicitSha256DigestComponent(digest)
			case "params-sha256":
				digest, err := hex.DecodeString(unescapedValue)
				if err != nil {
					return nil, errors.New("ParametersSha256DigestComponent is not a hex string")
				}
				c = NewParametersSha256DigestComponent(digest)
			case "8":
				c = NewGenericNameComponent([]byte(unescapedValue))
			case "seg":
				seg, err := strconv.ParseUint(unescapedValue, 10, 64)
				if err != nil {
					return nil, errors.New("SegmentNameComponent is not a decimal string")
				}
				c = NewSegmentNameComponent(seg)
			case "off":
				off, err := strconv.ParseUint(unescapedValue, 10, 64)
				if err != nil {
					return nil, errors.New("ByteOffsetNameComponent is not a decimal string")
				}
				c = NewByteOffsetNameComponent(off)
			case "v":
				v, err := strconv.ParseUint(unescapedValue, 10, 64)
				if err != nil {
					return nil, errors.New("VersionNameComponent is not a decimal string")
				}
				c = NewVersionNameComponent(v)
			case "t":
				t, err := strconv.ParseUint(unescapedValue, 10, 64)
				if err != nil {
					return nil, errors.New("TimestampNameComponent is not a decimal string")
				}
				c = NewTimestampNameComponent(t)
			case "seq":
				seq, err := strconv.ParseUint(unescapedValue, 10, 64)
				if err != nil {
					return nil, errors.New("SequenceNumNameComponent is not a decimal string")
				}
				c = NewSequenceNumNameComponent(seq)
			default:
				t, err := strconv.ParseUint(componentSplit[0], 10, 16)
				if err != nil {
					return nil, errors.New("unable to decode component type \"" + componentSplit[0] + "\"")
				}
				c = NewBaseNameComponent(uint16(t), []byte(unescapedValue))
				//return nil, errors.New("unknown name component " + unescapedValue)
			}
		} else {
			// Treat as GenericNameComponent
			unescaped, err := unescapeComponent(component)
			if err != nil {
				return nil, errors.New("error unescaping component value")
			}
			c = NewGenericNameComponent([]byte(unescaped))
		}
		n.Append(c)
	}

	return n, nil
}

func escapeComponent(in []byte) string {
	out := make([]byte, 0, 3*len(in)) // Capacity of 3 * len is worst case if every character has to be escaped
	nPeriods := 0
	for _, b := range in {
		switch {
		case b == '.':
			nPeriods++
			fallthrough
		case (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-' || b == '_' || b == '~':
			out = append(out, b)
		default:
			out = append(out, '%', 0, 0)
			hex.Encode(out[len(out)-2:], []byte{b})
		}
	}
	if nPeriods == len(in) {
		out = append(out, '.', '.', '.')
	}
	return string(out)
}

func unescapeComponent(in string) (string, error) {
	out := make([]byte, 0, len(in)) // Capacity is worst case if nothing to be unescaped
	for i := 0; i < len(in); i++ {
		if in[i] == '%' {
			// Unescape
			if len(in) <= i+2 {
				return "", errors.New("incomplete escape sequence")
			}
			unescaped, err := hex.DecodeString(in[i+1 : i+3])
			if err != nil {
				return "nil", errors.New("could not decode escape sequence")
			}
			out = append(out, unescaped...)
			i += 2
		} else {
			out = append(out, in[i])
		}
	}
	return string(out), nil
}

// DecodeName decodes a name from wire encoding.,
func DecodeName(b *tlv.Block) (*Name, error) {
	if b == nil {
		return nil, util.ErrNonExistent
	}
	_, err := b.Wire()
	if err != nil {
		return nil, err
	}
	if b.Type() != tlv.Name {
		return nil, tlv.ErrUnexpected
	}

	if len(b.Subelements()) == 0 {
		b.Parse()
	}
	n := new(Name)
	n.components = make([]NameComponent, len(b.Subelements()))
	for i, elem := range b.Subelements() {
		component, err := DecodeNameComponent(elem)
		if err != nil {
			return nil, err
		}
		n.components[i] = component
		n.cachedString += "/" + component.String()
	}
	n.wire = b
	return n, nil
}

func (n *Name) String() string {
	if len(n.cachedString) > 0 {
		return n.cachedString
	}

	if n.Size() == 0 {
		return "/"
	}

	var out string
	for _, component := range n.components {
		out += "/" + component.String()
	}
	n.cachedString = out
	return out
}

// Append adds the specified name component to the end of the name.
func (n *Name) Append(component NameComponent) *Name {
	n.components = append(n.components, component)
	n.wire = nil
	n.cachedString += "/" + component.String()
	return n
}

// At returns the name component at the specified index. If out of range, nil is returned.
func (n *Name) At(index int) NameComponent {
	if index < -len(n.components) || index >= len(n.components) {
		return nil
	}

	if index < 0 {
		return n.components[len(n.components)+index]
	}
	return n.components[index]
}

// Clear erases all components from the name.
func (n *Name) Clear() {
	if len(n.components) > 0 {
		n.components = make([]NameComponent, 0)
		n.wire = nil
		n.cachedString = ""
	}
}

// Compare returns the canonical order of this name against the the specified other name.
func (n *Name) Compare(other *Name) int {
	if n.Equals(other) {
		return 0
	} else if n.PrefixOf(other) {
		return -1
	} else if other.PrefixOf(n) {
		return 1
	}

	for i := 0; i < int(math.Min(float64(n.Size()), float64(other.Size()))); i++ {
		if n.At(i).Type() < other.At(i).Type() {
			return -1
		} else if n.At(i).Type() > other.At(i).Type() {
			return 1
		} else if len(n.At(i).Value()) < len(other.At(i).Value()) {
			return -1
		} else if len(n.At(i).Value()) > len(other.At(i).Value()) {
			return 1
		} else if !bytes.Equal(n.At(i).Value(), other.At(i).Value()) {
			// Do byte-by-byte comparison
			nValue := n.At(i).Value()
			otherValue := other.At(i).Value()
			for j := 0; j < len(nValue); j++ {
				if nValue[j] < otherValue[j] {
					return -1
				} else if nValue[j] > otherValue[j] {
					return 1
				}
			}
		}
	}

	// The only possibility left is that they exactly match.
	return 0
}

// DeepCopy returns a deep copy of the name.
func (n *Name) DeepCopy() *Name {
	name := new(Name)
	name.components = make([]NameComponent, 0)
	for _, component := range n.components {
		name.components = append(name.components, component.DeepCopy())
	}
	name.wire = nil
	name.cachedString = ""
	return name
}

// Equals returns whether the specified name is equal to this name.
func (n *Name) Equals(other *Name) bool {
	if n.Size() != other.Size() {
		return false
	}

	for i := 0; i < n.Size(); i++ {
		if n.At(i).Type() != other.At(i).Type() || !bytes.Equal(n.At(i).Value(), other.At(i).Value()) {
			return false
		}
	}

	return true
}

// Erase erases the specified name component. If out of range, no action is taken.
func (n *Name) Erase(index int) error {
	if index < 0 || index >= len(n.components) {
		return util.ErrOutOfRange
	}

	copy(n.components[index:], n.components[index+1:])
	n.components = n.components[:len(n.components)-1]
	n.wire = nil
	n.cachedString = ""
	return nil
}

// Find returns the first name component with the specified type, as well as its index.
func (n *Name) Find(tlvType uint16) (int, NameComponent) {
	for i, component := range n.components {
		if component.Type() == tlvType {
			return i, component
		}
	}

	return -1, nil
}

// HasWire returns whether the name has a wire encoding.
func (n *Name) HasWire() bool {
	return n.wire != nil
}

// Insert inserts a name component at the specified index.
func (n *Name) Insert(index int, component NameComponent) error {
	if index < 0 || index >= n.Size() {
		return util.ErrOutOfRange
	}

	n.components = append(n.components[:index], append([]NameComponent{component.DeepCopy()}, n.components[index:]...)...)
	n.wire = nil
	n.cachedString = ""
	return nil
}

// Prefix returns a name prefix of the specified number of components. If greater than or equal to the size of the name, this returns a copy of the name.
func (n *Name) Prefix(size int) *Name {
	prefix := *n
	prefix.components = make([]NameComponent, 0, len(n.components))
	for i := 0; i < size; i++ {
		// We have to deep copy this
		//prefix.components = append(prefix.components, reflect.New(reflect.ValueOf(component).Elem().Type()).Interface().(NameComponent))
		prefix.components = append(prefix.components, n.components[i].DeepCopy())
	}
	// Reset wire
	prefix.wire = new(tlv.Block)
	return &prefix
}

// PrefixOf returns whether this name is a prefix of the specified name.
func (n *Name) PrefixOf(other *Name) bool {
	if other == nil || n.Size() > other.Size() {
		return false
	}

	for i := 0; i < n.Size(); i++ {
		if n.At(i).Type() != other.At(i).Type() || !bytes.Equal(n.At(i).Value(), other.At(i).Value()) {
			return false
		}
	}

	return true
}

// Set replaces the component at the specified index with the specified component.
func (n *Name) Set(index int, component NameComponent) error {
	if index < 0 || index >= len(n.components) {
		return util.ErrOutOfRange
	}

	//n.components[index] = reflect.New(reflect.ValueOf(component).Elem().Type()).Interface().(NameComponent)
	n.components[index] = component
	n.wire = nil
	n.cachedString = ""
	return nil
}

// Size returns the number of components in the name.
func (n *Name) Size() int {
	return len(n.components)
}

// Encode encodes the name into a bock.
func (n *Name) Encode() *tlv.Block {
	if n.wire == nil {
		n.wire = new(tlv.Block)
		n.wire.SetType(tlv.Name)

		for _, component := range n.components {
			n.wire.Append(component.Encode())
		}

		n.wire.Wire()
	}
	return n.wire
}
