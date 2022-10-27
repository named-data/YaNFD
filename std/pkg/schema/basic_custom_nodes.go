// Basic custom nodes for test and demo use
package schema

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
)

// ContentKeyNode is a proof-of-concept demo to show how NTSchema can support NAC
// For simplicity we don't use KEK and KDK here.
type ContentKeyNode struct {
	BaseNode

	leaf *LeafNode
}

func (n *ContentKeyNode) Init(parent NTNode, edge enc.ComponentPattern) {
	n.BaseNode.Init(parent, edge)

	pat, _ := enc.NamePatternFromStr("/<ckid>")
	n.leaf = &LeafNode{}
	n.PutNode(pat, n.leaf)

	// The following setting is simply a default. One may overwrite it by policies after construction of the schema tree.
	n.leaf.Set(PropCanBePrefix, false)
	n.leaf.Set(PropMustBeFresh, false)
	n.leaf.Set(PropFreshness, 876000*time.Hour)
	n.leaf.Set(PropValidDuration, 876000*time.Hour)
	n.leaf.Set(PropDataSigner, sec.NewSha256Signer())
	passAllChecker := func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, Context) ValidRes {
		return VrPass
	}
	AddEventListener(n.leaf, PropOnValidateData, passAllChecker)

	n.Self = n
}

func (n *ContentKeyNode) GenKey(matching enc.Matching) []byte {
	keybits := make([]byte, 32)
	rand.Read(keybits) // Should always succeed
	ckid := make([]byte, 8)
	rand.Read(ckid)
	matching["ckid"] = ckid
	// Produce the key. Storage policies will decide where to store the key
	n.leaf.Provide(matching, nil, enc.Wire{keybits}, Context{})
	return ckid
}

func (n *ContentKeyNode) Encrypt(matching enc.Matching, ckid []byte, content enc.Wire) (enc.Wire, error) {
	if len(ckid) != 8 {
		return nil, fmt.Errorf("invalid content key id: %v", hex.EncodeToString(ckid))
	}
	matching["ckid"] = ckid
	res, keyWire := n.leaf.Need(matching, nil, nil, Context{CkSupressInt: true})
	if res != ndn.InterestResultData {
		return nil, fmt.Errorf("unable to get required content key for id: %v", hex.EncodeToString(ckid))
	}
	aescis, err := aes.NewCipher(keyWire.Join())
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aescis.BlockSize())
	rand.Read(iv)
	cip := cipher.NewCBCEncrypter(aescis, iv)
	bs := cip.BlockSize()
	l := content.Length()
	blkn := (int(l) + bs - 1) / bs
	totalLen := 8 + 8 + bs + blkn*bs // Use TLV in real world
	outbuf := make([]byte, totalLen)
	inbuf := make([]byte, blkn*bs)
	bp := 0
	for _, v := range content {
		bp += copy(inbuf[bp:], v)
	}
	binary.LittleEndian.PutUint64(outbuf[0:8], l)
	copy(outbuf[8:8+8], ckid)
	copy(outbuf[8+8:8+8+bs], iv)
	cip.CryptBlocks(outbuf[8+8+bs:], inbuf)
	return enc.Wire{outbuf}, nil
}

func (n *ContentKeyNode) Decrypt(matching enc.Matching, content enc.Wire) (enc.Wire, error) {
	buf := content.Join()
	l := binary.LittleEndian.Uint64(buf[0:8])
	ckid := buf[8 : 8+8]
	matching["ckid"] = ckid
	res, keyWire := n.leaf.Need(matching, nil, nil, Context{})
	if res != ndn.InterestResultData {
		return nil, fmt.Errorf("unable to get required content key for id: %v", hex.EncodeToString(ckid))
	}
	aescis, err := aes.NewCipher(keyWire.Join())
	if err != nil {
		return nil, err
	}

	bs := aescis.BlockSize()
	iv := buf[8+8 : 8+8+bs]
	inbuf := buf[8+8+bs:]
	cip := cipher.NewCBCDecrypter(aescis, iv)
	if len(inbuf)%cip.BlockSize() != 0 {
		return nil, fmt.Errorf("input AES buf has a wrong length")
	}
	cip.CryptBlocks(inbuf, inbuf)
	return enc.Wire{inbuf[:l]}, nil
}

// TODO: Not related but somehow the CK Name string() contains non-alphabet
