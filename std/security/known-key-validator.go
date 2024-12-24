package security

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// HmacValidate verifies the sha256 digest.
func Sha256Validate(sigCovered enc.Wire, sig ndn.Signature) bool {
	if sig.SigType() != ndn.SignatureDigestSha256 {
		return false
	}
	h := sha256.New()
	for _, buf := range sigCovered {
		_, err := h.Write(buf)
		if err != nil {
			return false
		}
	}
	return bytes.Equal(h.Sum(nil), sig.SigValue())
}

// HmacValidate verifies the signature with a known HMAC shared key.
func HmacValidate(sigCovered enc.Wire, sig ndn.Signature, key []byte) bool {
	if sig.SigType() != ndn.SignatureHmacWithSha256 {
		return false
	}
	return CheckHmacSig(sigCovered, sig.SigValue(), key)
}

// EcdsaValidate verifies the signature with a known ECC public key.
// ndn-cxx's PIB uses secp256r1 key stored in ASN.1 DER format. Use x509.ParsePKIXPublicKey to parse.
func EcdsaValidate(sigCovered enc.Wire, sig ndn.Signature, pubKey *ecdsa.PublicKey) bool {
	if sig.SigType() != ndn.SignatureSha256WithEcdsa {
		return false
	}
	h := sha256.New()
	for _, buf := range sigCovered {
		_, err := h.Write(buf)
		if err != nil {
			return false
		}
	}
	digest := h.Sum(nil)
	return ecdsa.VerifyASN1(pubKey, digest, sig.SigValue())
}

// RsaValidate verifies the signature with a known RSA public key.
// ndn-cxx's PIB uses RSA 2048 key stored in ASN.1 DER format. Use x509.ParsePKIXPublicKey to parse.
func RsaValidate(sigCovered enc.Wire, sig ndn.Signature, pubKey *rsa.PublicKey) bool {
	if sig.SigType() != ndn.SignatureSha256WithRsa {
		return false
	}
	h := sha256.New()
	for _, buf := range sigCovered {
		_, err := h.Write(buf)
		if err != nil {
			return false
		}
	}
	digest := h.Sum(nil)
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest, sig.SigValue()) == nil
}

// EddsaValidate verifies the signature with a known ed25519 public key.
// ndn-cxx's PIB does not support this, but a certificate is supposed to use ASN.1 DER format.
// Use x509.ParsePKIXPublicKey to parse. Note: ed25519.PublicKey is defined to be a pointer type without '*'.
func EddsaValidate(sigCovered enc.Wire, sig ndn.Signature, pubKey ed25519.PublicKey) bool {
	if sig.SigType() != ndn.SignatureEd25519 {
		return false
	}
	return ed25519.Verify(pubKey, sigCovered.Join(), sig.SigValue())
}
