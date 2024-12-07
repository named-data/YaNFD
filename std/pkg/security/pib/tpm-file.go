package sqlitepib

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

type FileTpm struct {
	path string
}

func (tpm *FileTpm) ToFileName(keyNameBytes []byte) string {
	h := sha256.New()
	h.Write(keyNameBytes)
	return hex.EncodeToString(h.Sum(nil)) + ".privkey"
}

func (tpm *FileTpm) GetSigner(keyName enc.Name, keyLocatorName enc.Name) ndn.Signer {
	keyNameBytes := keyName.Bytes()
	fileName := path.Join(tpm.path, tpm.ToFileName(keyNameBytes))

	text, err := os.ReadFile(fileName)
	if err != nil {
		log.WithField("module", "FileTpm").Errorf("unable to read private key file: %s, %+v", fileName, err)
		return nil
	}

	blockLen := base64.StdEncoding.DecodedLen(len(text))
	block := make([]byte, blockLen)
	n, err := base64.StdEncoding.Decode(block, text)
	if err != nil {
		log.WithField("module", "FileTpm").Errorf("unable to base64 decode private key file: %s, %+v", fileName, err)
		return nil
	}
	block = block[:n]

	// There are only two formats: PKCS1 encoded RSA, or EC
	eckbits, err := x509.ParseECPrivateKey(block)
	if err == nil {
		// ECC Key
		// TODO: Handle for Interest
		return sec.NewEccSigner(false, false, 0, eckbits, keyLocatorName)
	}

	rsabits, err := x509.ParsePKCS1PrivateKey(block)
	if err == nil {
		// RSA Key
		// TODO: Handle for Interest
		return sec.NewRsaSigner(false, false, 0, rsabits, keyLocatorName)
	}

	log.WithField("module", "FileTpm").Errorf("unrecognized private key format: %s", fileName)
	return nil
}

func (tpm *FileTpm) GenerateKey(keyName enc.Name, keyType string, keySize uint64) enc.Buffer {
	panic("not implemented")
}

func (tpm *FileTpm) KeyExist(keyName enc.Name) bool {
	keyNameBytes := keyName.Bytes()
	fileName := path.Join(tpm.path, tpm.ToFileName(keyNameBytes))
	_, err := os.Stat(fileName)
	return err == nil
}

func (tpm *FileTpm) DeleteKey(keyName enc.Name) {
	panic("not implemented")
}

func NewFileTpm(path string) Tpm {
	return &FileTpm{
		path: path,
	}
}
