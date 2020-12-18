/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package security

import (
	"errors"
)

// SignatureType represents the type of a signature.
type SignatureType uint64

// The various possible values of SignatureType.
const (
	DigestSha256Type             SignatureType = 0
	SignatureSha256WithRsaType   SignatureType = 1
	SignatureSha256WithEcdsaType SignatureType = 3
	SignatureHmacWithSha256Type  SignatureType = 4
)

// Signer represents an implementation of a signature type.
type Signer interface {
	Sign(buffer []byte) ([]byte, error)
	Validate(buffer []byte, signature []byte) bool
}

// Sign signs the provided buffer using the appropriate signer.
func Sign(signatureType SignatureType, buffer []byte) ([]byte, error) {
	switch signatureType {
	case DigestSha256Type:
		var signer DigestSha256
		signature, err := signer.Sign(buffer)
		if err != nil {
			return nil, err
		}
		return signature, nil
	case SignatureSha256WithRsaType:
		return nil, errors.New("Cannot sign SignatureSha256WithRsa")
	case SignatureSha256WithEcdsaType:
		return nil, errors.New("Cannot sign SignatureSha256WithEcdsaType")
	case SignatureHmacWithSha256Type:
		return nil, errors.New("Cannot sign SignatureHmacWithSha256Type")
	default:
		return nil, errors.New("Unknown SignatureType")
	}
}

// Verify verifies the provided signature against the provided buffer using the appropriate signer.
func Verify(signatureType SignatureType, buffer []byte, signature []byte) (bool, error) {
	switch signatureType {
	case DigestSha256Type:
		var signer DigestSha256
		return signer.Validate(buffer, signature), nil
	case SignatureSha256WithRsaType:
		return false, errors.New("Cannot validate SignatureSha256WithRsa")
	case SignatureSha256WithEcdsaType:
		return false, errors.New("Cannot validate SignatureSha256WithEcdsaType")
	case SignatureHmacWithSha256Type:
		return false, errors.New("Cannot validate SignatureHmacWithSha256Type")
	default:
		// Unknown SignatureType
		return false, errors.New("Unknown SignatureType")
	}
}
