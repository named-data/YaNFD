package sqlitepib

import (
	enc "github.com/pulsejet/ndnd/std/encoding"
	"github.com/pulsejet/ndnd/std/ndn"
)

// Tpm is a sceure storage that holds the private key
type Tpm interface {
	GetSigner(keyName enc.Name, keyLocatorName enc.Name) ndn.Signer
	GenerateKey(keyName enc.Name, keyType string, keySize uint64) enc.Buffer
	KeyExist(keyName enc.Name) bool
	DeleteKey(keyName enc.Name)
}

// Cert represents a certificate one owns
type Cert interface {
	AsSigner() ndn.Signer
	Name() enc.Name
	Key() Key
	Data() []byte
	// KeyLocator is the name of the key/certificate which signs this certificate
	KeyLocator() enc.Name
}

// Key represents a key one owns (with both private and public keybits)
type Key interface {
	Name() enc.Name
	Identity() Identity
	KeyBits() []byte
	SelfSignedCert() Cert
	GetCert(enc.Name) Cert
	FindCert(func(Cert) bool) Cert
}

// Identity represents an identity one owns
type Identity interface {
	Name() enc.Name
	GetKey(enc.Name) Key
	FindCert(func(Cert) bool) Cert
}

// Pib is a storage storing all owned identities, keys and certificates.
type Pib interface {
	Tpm() Tpm
	GetIdentity(name enc.Name) Identity
}
