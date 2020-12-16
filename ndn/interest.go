/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/ndn/util"
)

// Interest represents an NDN Interest packet.
type Interest struct {
	name           Name
	canBePrefix    bool
	mustBeFresh    bool
	forwardingHint []Delegation
	nonce          []byte
	lifetime       time.Duration
	hopLimit       *uint8
	parameters     []*tlv.Block
	wire           *tlv.Block
}

// NewInterest creates a new Interest with the specified name and default values.
func NewInterest(name *Name) *Interest {
	i := new(Interest)
	i.name = *name.DeepCopy()
	i.lifetime = 4000 * time.Millisecond
	i.ResetNonce()
	return i
}

// DecodeInterest decodes an Interest from the wire.
func DecodeInterest(wire *tlv.Block) (*Interest, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}
	wire.Parse()

	i := new(Interest)
	i.wire = wire.DeepCopy()
	mostRecentElem := 0
	hasApplicationParameters := false
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case tlv.Name:
			if mostRecentElem >= 1 {
				return nil, errors.New("Name is duplicate or out-of-order")
			}
			name, err := DecodeName(elem)
			if err != nil {
				return nil, err
			}
			mostRecentElem = 1
			i.SetName(name)
		case tlv.CanBePrefix:
			if mostRecentElem >= 2 {
				return nil, errors.New("CanBePrefix is duplicate or out-of-order")
			}
			mostRecentElem = 2
			i.SetCanBePrefix(true)
		case tlv.MustBeFresh:
			if mostRecentElem >= 3 {
				return nil, errors.New("MustBeFresh is duplicate or out-of-order")
			}
			mostRecentElem = 3
			i.SetMustBeFresh(true)
		case tlv.ForwardingHint:
			if mostRecentElem >= 4 {
				return nil, errors.New("ForwardingHint is duplicate or out-of-order")
			}
			mostRecentElem = 4
			elem.Parse()
			for _, delegationBlock := range elem.Subelements() {
				delegation, err := DecodeDelegation(delegationBlock)
				if err != nil {
					return nil, errors.New("Error decoding Delegation")
				}
				i.forwardingHint = append(i.forwardingHint, *delegation)
			}
		case tlv.Nonce:
			if mostRecentElem >= 5 {
				return nil, errors.New("Nonce is duplicate or out-of-order")
			}
			mostRecentElem = 5
			if i.SetNonce(elem.Value()) != nil {
				return nil, errors.New("Error decoding Nonce")
			}
		case tlv.InterestLifetime:
			if mostRecentElem >= 6 {
				return nil, errors.New("InterestLifetime is duplicate or out-of-order")
			}
			mostRecentElem = 6
			lifetime, err := tlv.DecodeNNIBlock(elem)
			if err != nil {
				return nil, errors.New("Error decoding InterestLifetime")
			}
			i.SetLifetime(time.Duration(lifetime) * time.Millisecond)
		case tlv.HopLimit:
			if mostRecentElem >= 7 {
				return nil, errors.New("HopLimit is duplicate or out-of-order")
			}
			mostRecentElem = 7
			if len(elem.Value()) != 1 {
				return nil, errors.New("Error decoding HopLimit")
			}
			i.SetHopLimit(&elem.Value()[0])
		case tlv.ApplicationParameters:
			if mostRecentElem >= 8 {
				return nil, errors.New("ApplicationParameters is duplicate or out-of-order")
			}
			mostRecentElem = 8
			hasApplicationParameters = true
			i.parameters = append(i.parameters, elem.DeepCopy())
		default:
			if !hasApplicationParameters && tlv.IsCritical(elem.Type()) {
				return nil, tlv.ErrUnrecognizedCritical
			} else if hasApplicationParameters {
				i.parameters = append(i.parameters, elem.DeepCopy())
			}
			// If non-critical and not after ApplicationParameters, ignore
		}
	}

	// If has ApplicationParameters, verify parameters digest component
	if hasApplicationParameters {
		_, paramsDigest := i.name.Find(tlv.ParametersSha256DigestComponent)
		if paramsDigest == nil {
			return nil, errors.New("Has ApplicationParameters but missing ParametersSha256DigestComponent")
		}
		// Hash parameters
		h := sha256.New()
		for _, param := range i.parameters {
			paramWire, err := param.Wire()
			if err != nil {
				return nil, errors.New("Error wire encoding application parameter of type 0x" + strconv.FormatUint(uint64(param.Type()), 16))
			}
			h.Write(paramWire)
		}
		generatedHash := h.Sum(nil)

		// Verify hash
		if !bytes.EqualFold(paramsDigest.Value(), generatedHash) {
			return nil, errors.New("ParametersSha256DigestComponent did not match hash of application parameters")
		}
	}

	return i, nil
}

func (i *Interest) String() string {
	str := "Interest(Name=" + i.name.String()

	if i.canBePrefix {
		str += ", CanBePrefix"
	}
	if i.mustBeFresh {
		str += ", MustBeFresh"
	}
	if len(i.forwardingHint) > 0 {
		str += ", ForwardingHint("
		isFirstDelegation := true
		for _, delegation := range i.forwardingHint {
			if !isFirstDelegation {
				str += ", "
			}
			str += delegation.String()
			isFirstDelegation = false
		}
		str += ")"
	}
	str += ", Nonce=0x" + hex.EncodeToString(i.nonce)
	str += ", Lifetime=" + strconv.FormatInt(i.lifetime.Milliseconds(), 10) + "ms"
	if i.hopLimit != nil {
		str += ", HopLimit=" + strconv.FormatUint(uint64(*i.hopLimit), 10)
	}
	if len(i.parameters) > 0 {
		str += ", ApplicationParameters"
	}

	str += ")"
	return str
}

//////////////////
// Setters/Getters
//////////////////

// Name returns a copy of the name of the Interest.
func (i *Interest) Name() *Name {
	return i.name.DeepCopy()
}

// SetName sets the name of the Interest.
func (i *Interest) SetName(name *Name) {
	i.name = *name.DeepCopy()
	i.wire = nil
}

// CanBePrefix returns whether the Interest can be satisfied by a Data packet whos name the Interest name is a prefix of.
func (i *Interest) CanBePrefix() bool {
	return i.canBePrefix
}

// SetCanBePrefix sets whether the Interest can be satisfied by a Data packet whos name the Interest name is a prefix of.
func (i *Interest) SetCanBePrefix(canBePrefix bool) {
	i.canBePrefix = canBePrefix
	i.wire = nil
}

// MustBeFresh returns whether the Interest can only be satisfied by "fresh" Data packets.
func (i *Interest) MustBeFresh() bool {
	return i.mustBeFresh
}

// SetMustBeFresh sets whether the Interest can only be satisfied by "fresh" Data packets.
func (i *Interest) SetMustBeFresh(mustBeFresh bool) {
	i.mustBeFresh = mustBeFresh
	i.wire = nil
}

// ForwardingHint returns a copy of the delegations in the ForwardingHint in the Interest.
func (i *Interest) ForwardingHint() []Delegation {
	if i.forwardingHint == nil {
		return make([]Delegation, 0)
	}

	fh := make([]Delegation, 0, len(i.forwardingHint))
	for _, delegation := range i.forwardingHint {
		fh = append(fh, delegation)
	}
	return fh
}

// AppendForwardingHint appends a delegation to the ForwardingHint in the Interest.
func (i *Interest) AppendForwardingHint(delegation *Delegation) {
	i.forwardingHint = append(i.forwardingHint, *delegation)
	i.wire = nil
}

// ClearForwardingHints removes all forwarding hints attached to the Interest.
func (i *Interest) ClearForwardingHints() {
	i.forwardingHint = []Delegation{}
	i.wire = nil
}

// EraseForwardingHint removes the forwarding hint at the specified index from the Interest.
func (i *Interest) EraseForwardingHint(index int) error {
	if index < 0 || index >= len(i.forwardingHint) {
		return util.ErrOutOfRange
	}

	i.forwardingHint = append(i.forwardingHint[:index], i.forwardingHint[index+1:]...)
	i.wire = nil
	return nil
}

// Nonce gets the nonce of the Interest.
func (i *Interest) Nonce() []byte {
	nonce := make([]byte, 4)
	copy(nonce, i.nonce)
	return nonce
}

// ResetNonce regenerates the value of the nonce.
func (i *Interest) ResetNonce() {
	i.nonce = make([]byte, 4)
	for pos := 0; pos < 4; pos++ {
		i.nonce[pos] = byte(rand.Uint32() % 256)
	}
	i.wire = nil
}

// SetNonce sets the nonce to the specified value. If not exactly 4 bytes, an error is returned.
func (i *Interest) SetNonce(nonce []byte) error {
	if len(nonce) != 4 {
		return util.ErrTooShort
	}

	i.nonce = make([]byte, 4)
	copy(i.nonce, nonce)
	i.wire = nil
	return nil
}

// Lifetime returns the lifetime of the Interest.
func (i *Interest) Lifetime() time.Duration {
	return i.lifetime
}

// SetLifetime set the lifetime of the Interest.
func (i *Interest) SetLifetime(lifetime time.Duration) {
	i.lifetime = lifetime
	i.wire = nil
}

// HopLimit returns the hop limit of the Interest or nil if no hop limit is set.
func (i *Interest) HopLimit() *uint8 {
	if i.hopLimit == nil {
		return nil
	}

	hopLimit := new(uint8)
	*hopLimit = *i.hopLimit
	return hopLimit
}

// SetHopLimit sets the hop limit of the Interest (or unsets it if nil is specified).
func (i *Interest) SetHopLimit(hopLimit *uint8) {
	if hopLimit == nil {
		i.hopLimit = nil
	} else {
		i.hopLimit = new(uint8)
		*i.hopLimit = *hopLimit
	}
	i.wire = nil
}

// ApplicationParameters returns a copy of the application parameters of the Interest.
func (i *Interest) ApplicationParameters() []tlv.Block {
	params := make([]tlv.Block, 0, len(i.parameters))
	for _, param := range i.parameters {
		params = append(params, *param.DeepCopy())
	}
	return params
}

// AppendApplicationParameter appends an application parameter to the Interest. If not already present (or the type of the parameter block specified), it adds an empty ApplicationParameters block before appending this block.
func (i *Interest) AppendApplicationParameter(block *tlv.Block) {
	if block.Type() != tlv.ApplicationParameters && len(i.parameters) == 0 {
		i.parameters = append(i.parameters, tlv.NewEmptyBlock(tlv.ApplicationParameters))
	}
	i.parameters = append(i.parameters, block.DeepCopy())

	// Reset ParametersDigestSha256Component
	i.recomputeParametersDigestComponent()

	i.wire = nil
}

func (i *Interest) recomputeParametersDigestComponent() {
	// Compute digest
	h := sha256.New()
	for _, param := range i.parameters {
		// We have verified no error
		paramWire, _ := param.Wire()
		h.Write(paramWire)
	}
	generatedHash := h.Sum(nil)

	digestIndex, _ := i.name.Find(tlv.ParametersSha256DigestComponent)
	if digestIndex != -1 {
		// Replace existing component
		i.name.Set(digestIndex, NewParametersSha256DigestComponent(generatedHash))
	} else {
		// Place according to ordering in spec (after last GenericNameComponent)
		lastGenericComponent := -1
		for ; lastGenericComponent < i.name.Size(); lastGenericComponent++ {
			if i.name.At(lastGenericComponent+1).Type() != tlv.GenericNameComponent {
				break
			}
		}

		if lastGenericComponent == -1 {
			// Insert at position 0
			i.name.Insert(0, NewParametersSha256DigestComponent(generatedHash))
		} else if lastGenericComponent == i.name.Size()-1 {
			// Append
			i.name.Append(NewParametersSha256DigestComponent(generatedHash))
		} else {
			// Insert after last GenericNameComponent
			i.name.Insert(lastGenericComponent+1, NewParametersSha256DigestComponent(generatedHash))
		}
	}

	i.wire = nil
}

// ClearApplicationParameters clears all ApplicationParameters from the Interest.
func (i *Interest) ClearApplicationParameters() {
	i.parameters = make([]*tlv.Block, 0)
	i.wire = nil
}

///////////
// Encoding
///////////

// Encode encodes the data into a block.
func (i *Interest) Encode() (*tlv.Block, error) {
	if i.wire != nil {
		return i.wire.DeepCopy(), nil
	}

	i.wire = new(tlv.Block)
	i.wire.SetType(tlv.Interest)

	// Validate fields
	if i.name.Size() == 0 {
		return nil, errors.New("Name cannot be empty")
	}

	if len(i.nonce) != 4 {
		return nil, errors.New("Nonce must be set to encode")
	}

	// Name
	i.wire.Append(i.name.Encode())

	// CanBePrefix
	if i.canBePrefix {
		i.wire.Append(tlv.NewEmptyBlock(tlv.CanBePrefix))
	}

	// MustBeFresh
	if i.mustBeFresh {
		i.wire.Append(tlv.NewEmptyBlock(tlv.MustBeFresh))
	}

	// ForwardingHint
	if len(i.forwardingHint) > 0 {
		fhBlock := tlv.NewEmptyBlock(tlv.ForwardingHint)
		for _, delegation := range i.forwardingHint {
			fhBlock.Append(delegation.Encode())
		}
		i.wire.Append(fhBlock)
	}

	// Nonce
	i.wire.Append(tlv.NewBlock(tlv.Nonce, i.nonce))

	// InterestLifetime
	i.wire.Append(tlv.EncodeNNIBlock(tlv.InterestLifetime, uint64(i.lifetime.Milliseconds())))

	// HopLimit
	if i.hopLimit != nil {
		i.wire.Append(tlv.NewBlock(tlv.HopLimit, []byte{*i.hopLimit}))
	}

	// ApplicationParameters
	for _, param := range i.parameters {
		i.wire.Append(param)
	}

	i.wire.Wire()
	return i.wire.DeepCopy(), nil
}

// HasWire returns whether a wire encoding exists for the Interest.
func (i *Interest) HasWire() bool {
	return i.wire != nil
}
