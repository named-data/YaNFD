package encoding

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	TypeInvalidComponent                TLNum = 0x00
	TypeImplicitSha256DigestComponent   TLNum = 0x01
	TypeParametersSha256DigestComponent TLNum = 0x02
	TypeGenericNameComponent            TLNum = 0x08
	TypeKeywordNameComponent            TLNum = 0x20
	TypeSegmentNameComponent            TLNum = 0x32
	TypeByteOffsetNameComponent         TLNum = 0x34
	TypeVersionNameComponent            TLNum = 0x36
	TypeTimestampNameComponent          TLNum = 0x38
	TypeSequenceNumNameComponent        TLNum = 0x3a
)

type compValFmt int

const (
	compValFmtInvalid compValFmt = iota
	compValFmtText
	compValFmtDec
	compValFmtHex
)

type componentConvention struct {
	typ  TLNum
	name string
	vFmt compValFmt
}

var (
	compConvByType = map[TLNum]*componentConvention{
		TypeImplicitSha256DigestComponent: {
			typ:  TypeImplicitSha256DigestComponent,
			name: "sha256digest",
			vFmt: compValFmtHex,
		},
		TypeParametersSha256DigestComponent: {
			typ:  TypeParametersSha256DigestComponent,
			name: "params-sha256",
			vFmt: compValFmtHex,
		},
		TypeSegmentNameComponent: {
			typ:  TypeSegmentNameComponent,
			name: "seg",
			vFmt: compValFmtDec,
		},
		TypeByteOffsetNameComponent: {
			typ:  TypeByteOffsetNameComponent,
			name: "off",
			vFmt: compValFmtDec,
		},
		TypeVersionNameComponent: {
			typ:  TypeVersionNameComponent,
			name: "v",
			vFmt: compValFmtDec,
		},
		TypeTimestampNameComponent: {
			typ:  TypeTimestampNameComponent,
			name: "t",
			vFmt: compValFmtDec,
		},
		TypeSequenceNumNameComponent: {
			typ:  TypeSequenceNumNameComponent,
			name: "seq",
			vFmt: compValFmtDec,
		},
	}
	compConvByStr map[string]*componentConvention
)

type ComponentPattern interface {
	// ComponentPatternTrait returns the type trait of Component or Pattern
	// This is used to make ComponentPattern a union type of Component or Pattern
	// Component | Pattern does not work because we need a mixed list NamePattern
	ComponentPatternTrait() ComponentPattern

	String() string

	Compare(ComponentPattern) int

	Equal(ComponentPattern) bool
}

type Pattern struct {
	Typ TLNum
	Tag string
}

type Component struct {
	Typ TLNum
	Val []byte
}

func (c *Component) ComponentPatternTrait() ComponentPattern {
	return c
}

func (p *Pattern) ComponentPatternTrait() ComponentPattern {
	return p
}

func initComponentConventions() {
	compConvByStr = make(map[string]*componentConvention, len(compConvByType))
	for _, c := range compConvByType {
		compConvByStr[c.name] = c
	}
}

func (p *Pattern) String() string {
	if p.Typ == TypeGenericNameComponent {
		return "<" + p.Tag + ">"
	} else if conv, ok := compConvByType[p.Typ]; ok {
		return "<" + conv.name + "=" + p.Tag + ">"
	} else {
		return fmt.Sprintf("<%d=%s>", p.Typ, p.Tag)
	}
}

func (c *Component) Length() TLNum {
	return TLNum(len(c.Val))
}

func isLegalCompText(b byte) bool {
	return unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b)) || b == '-' || b == '_' || b == '.' || b == '~'
}

func (c *Component) String() string {
	vFmt := compValFmtText
	tName := ""
	if conv, ok := compConvByType[c.Typ]; ok {
		vFmt = conv.vFmt
		tName = conv.name + "="
	} else if c.Typ != TypeGenericNameComponent {
		tName = strconv.FormatUint(uint64(c.Typ), 10) + "="
	}
	vText := ""
	switch vFmt {
	case compValFmtDec:
		x := uint64(0)
		for _, b := range c.Val {
			x = (x << 8) | uint64(b)
		}
		vText = strconv.FormatUint(x, 10)
	case compValFmtHex:
		for _, b := range c.Val {
			vText = vText + fmt.Sprintf("%02x", b)
		}
	case compValFmtText:
		for _, b := range c.Val {
			if isLegalCompText(b) {
				vText = vText + string(b)
			} else {
				vText = vText + fmt.Sprintf("%%%02X", b)
			}
		}
	}
	return tName + vText
}

func ParseComponent(buf Buffer) (Component, int) {
	typ, p1 := ParseTLNum(buf)
	l, p2 := ParseTLNum(buf[p1:])
	start := p1 + p2
	end := start + int(l)
	return Component{
		Typ: typ,
		Val: buf[start:end],
	}, end
}

func ReadComponent(r ParseReader) (*Component, error) {
	typ, err := ReadTLNum(r)
	if err != nil {
		return nil, err
	}
	l, err := ReadTLNum(r)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	val, err := r.ReadWire(int(l))
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if len(val) == 0 {
		return &Component{
			Typ: typ,
			Val: []byte{},
		}, nil
	} else if len(val) == 1 {
		// If it is within one fragment, no copy
		return &Component{
			Typ: typ,
			Val: val[0],
		}, nil
	} else {
		// If it crosses fragment boundary (very rare), copy it
		valBuf := make([]byte, int(l))
		pos := 0
		for _, v := range val {
			copy(valBuf[pos:pos+len(v)], v)
			pos += len(v)
		}
		return &Component{
			Typ: typ,
			Val: valBuf,
		}, nil
	}
}

func parseCompTypeFromStr(s string) (TLNum, compValFmt, error) {
	if unicode.IsLetter(rune(s[0])) {
		if conv, ok := compConvByStr[s]; ok {
			return conv.typ, conv.vFmt, nil
		} else {
			return 0, 0, ErrFormat{"unknown component type: " + s}
		}
	} else {
		typInt, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return 0, 0, ErrFormat{"invalid component type: " + s}
		}
		return TLNum(typInt), compValFmtText, nil
	}
}

func ComponentFromStr(s string) (*Component, error) {
	var err error
	strs := strings.Split(s, "=")
	if len(strs) > 2 {
		return nil, ErrFormat{"too many '=' in component: " + s}
	}
	valStr := strs[len(strs)-1]
	typ := TypeGenericNameComponent
	vFmt := compValFmtText
	val := []byte(nil)
	if len(strs) == 2 {
		typ, vFmt, err = parseCompTypeFromStr(strs[0])
		if err != nil {
			return nil, err
		}
		if typ <= TypeInvalidComponent || int(typ) > 0xffff {
			return nil, ErrFormat{"invalid component type: " + valStr}
		}
	}
	switch vFmt {
	case compValFmtDec:
		x, err := strconv.ParseUint(valStr, 10, 64)
		if err != nil {
			return nil, ErrFormat{"invalid decimal component value: " + valStr}
		}
		val = make([]byte, Nat(x).EncodingLength())
		Nat(x).EncodeInto(val)
	case compValFmtHex:
		if len(valStr)%2 != 0 {
			return nil, ErrFormat{"invalid hexadecimal component value: " + valStr}
		}
		l := len(valStr) / 2
		val = make([]byte, l)
		for i := 0; i < l; i++ {
			b, err := strconv.ParseUint(valStr[i*2:i*2+2], 16, 8)
			if err != nil {
				return nil, ErrFormat{"invalid hexadecimal component value: " + valStr}
			}
			val[i] = byte(b)
		}
	case compValFmtText:
		val = make([]byte, 0)
		for i := 0; i < len(valStr); {
			if isLegalCompText(valStr[i]) {
				val = append(val, valStr[i])
				i++
			} else if valStr[i] == '%' && i+2 < len(valStr) {
				v, err := strconv.ParseUint(valStr[i+1:i+3], 16, 8)
				if err != nil {
					return nil, ErrFormat{"invalid component value: " + valStr}
				}
				val = append(val, byte(v))
				i += 3
			} else {
				// Gracefully accept invalid character
				if valStr[i] != '%' && valStr[i] != '=' && valStr[i] != '/' && valStr[i] != '\\' {
					val = append(val, valStr[i])
					i++
				} else {
					return nil, ErrFormat{"invalid component value: " + valStr}
				}
			}
		}
	default:
		panic("unknown component value format")
	}
	return &Component{
		Typ: typ,
		Val: val,
	}, nil
}

func ComponentPatternFromStr(s string) (ComponentPattern, error) {
	if len(s) <= 0 || s[0] == '<' {
		return ComponentFromStr(s)
	}
	if s[len(s)-1] != '>' {
		return nil, ErrFormat{"invalid component pattern: " + s}
	}
	s = s[1 : len(s)-1]
	strs := strings.Split(s, "=")
	if len(strs) > 2 {
		return nil, ErrFormat{"too many '=' in component pattern: " + s}
	}
	if len(strs) == 2 {
		typ, _, err := parseCompTypeFromStr(strs[0])
		if err != nil {
			return nil, err
		}
		return &Pattern{
			Typ: typ,
			Tag: strs[1],
		}, nil
	} else {
		return &Pattern{
			Typ: TypeGenericNameComponent,
			Tag: strs[0],
		}, nil
	}
}

func (c *Component) EncodingLength() int {
	l := len(c.Val)
	return c.Typ.EncodingLength() + Nat(l).EncodingLength() + l
}

func (c *Component) EncodeInto(buf Buffer) int {
	p1 := c.Typ.EncodeInto(buf)
	p2 := Nat(len(c.Val)).EncodeInto(buf[p1:])
	copy(buf[p1+p2:], c.Val)
	return p1 + p2 + len(c.Val)
}

func (c *Component) Bytes() []byte {
	buf := make([]byte, c.EncodingLength())
	c.EncodeInto(buf)
	return buf
}

func ComponentFromBytes(buf []byte) (*Component, error) {
	r := NewBufferReader(buf)
	return ReadComponent(r)
}

func (c *Component) Compare(rhs ComponentPattern) int {
	rc, ok := rhs.(*Component)
	if !ok {
		return -1
	}
	if c.Typ != rc.Typ {
		if c.Typ < rc.Typ {
			return -1
		} else {
			return 1
		}
	}
	if len(c.Val) != len(rc.Val) {
		if len(c.Val) < len(rc.Val) {
			return -1
		} else {
			return 1
		}
	}
	return bytes.Compare(c.Val, rc.Val)
}

func (c *Component) Equal(rhs ComponentPattern) bool {
	rc, ok := rhs.(*Component)
	if !ok {
		return false
	}
	if c.Typ != rc.Typ || len(c.Val) != len(rc.Val) {
		return false
	}
	return bytes.Equal(c.Val, rc.Val)
}

func (p *Pattern) Compare(rhs ComponentPattern) int {
	rp, ok := rhs.(*Pattern)
	if !ok {
		return 1
	}
	if p.Typ != rp.Typ {
		if p.Typ < rp.Typ {
			return -1
		} else {
			return 1
		}
	}
	return strings.Compare(p.Tag, rp.Tag)
}

func (p *Pattern) Equal(rhs ComponentPattern) bool {
	rp, ok := rhs.(*Pattern)
	if !ok {
		return false
	}
	return p.Typ == rp.Typ && p.Tag == rp.Tag
}

func NewNumberComponent(typ TLNum, val uint64) *Component {
	return &Component{
		Typ: typ,
		Val: Nat(val).Bytes(),
	}
}

func NewSegmentComponent(seg uint64) *Component {
	return NewNumberComponent(TypeSegmentNameComponent, seg)
}

func NewByteOffsetComponent(off uint64) *Component {
	return NewNumberComponent(TypeByteOffsetNameComponent, off)
}

func NewSequenceNumComponent(seq uint64) *Component {
	return NewNumberComponent(TypeSequenceNumNameComponent, seq)
}

func NewVersionComponent(v uint64) *Component {
	return NewNumberComponent(TypeVersionNameComponent, v)
}

func NewTimestampComponent(t uint64) *Component {
	return NewNumberComponent(TypeTimestampNameComponent, t)
}

func NewBytesComponent(typ TLNum, val []byte) *Component {
	return &Component{
		Typ: typ,
		Val: val,
	}
}

func NewStringComponent(typ TLNum, val string) *Component {
	return &Component{
		Typ: typ,
		Val: []byte(val),
	}
}
