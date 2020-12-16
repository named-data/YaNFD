/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"strconv"

	"github.com/eric135/YaNFD/ndn/tlv"
	"github.com/eric135/YaNFD/ndn/util"
)

// Delegation contains a Link Object delegation.
type Delegation struct {
	preference uint64
	name       Name
	wire       tlv.Block
}

// NewDelegation creates a new delegation.
func NewDelegation(preference uint64, name *Name) (*Delegation, error) {
	d := new(Delegation)
	d.SetPreference(preference)
	d.SetName(name)
	return d, nil
}

// DecodeDelegation decodes a delegation from the wire.
func DecodeDelegation(wire *tlv.Block) (*Delegation, error) {
	if wire == nil {
		return nil, util.ErrNonExistent
	}
	wire.Parse()
	if wire.Find(tlv.Preference) == nil || wire.Find(tlv.Name) == nil {
		return nil, util.ErrNonExistent
	}

	d := new(Delegation)
	var err error
	d.preference, err = tlv.DecodeNNIBlock(wire.Find(tlv.Preference))
	if err != nil {
		return nil, err
	}

	name, err := DecodeName(wire.Find(tlv.Name))
	if err != nil {
		return nil, err
	}
	d.name = *name

	d.wire = *wire.DeepCopy()
	return d, nil
}

func (d *Delegation) String() string {
	return "Delegation(" + strconv.FormatUint(d.preference, 10) + ", " + d.name.String() + ")"
}

// DeepCopy returns a deep copy of the delegation.
func (d *Delegation) DeepCopy() *Delegation {
	copyD := new(Delegation)
	copyD.preference = d.preference
	copyD.name = *d.name.DeepCopy()
	return copyD
}

// Preference returns the preference set in the delegation.
func (d *Delegation) Preference() uint64 {
	return d.preference
}

// SetPreference sets the preference in the delegation.
func (d *Delegation) SetPreference(preference uint64) {
	d.preference = preference
	d.wire.Reset()
}

// Name returns a copy of the name set in the delegation.
func (d *Delegation) Name() *Name {
	return d.name.DeepCopy()
}

// SetName sets the name in the delegation.
func (d *Delegation) SetName(name *Name) {
	d.name = *name.DeepCopy()
	d.wire.Reset()
}

// Encode encodes the delegation into a block.
func (d *Delegation) Encode() *tlv.Block {
	if !d.wire.HasWire() {
		d.wire.SetType(tlv.Delegation)
		d.wire.Append(tlv.EncodeNNIBlock(tlv.Preference, d.preference))
		d.wire.Append(d.name.Encode())
		d.wire.Wire()
	}
	return d.wire.DeepCopy()
}
