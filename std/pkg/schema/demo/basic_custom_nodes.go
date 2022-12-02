// Basic custom nodes for test and demo use. Not secure for production.
package demo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// ContentKeyNode is a proof-of-concept demo to show how NTSchema can support NAC
// For simplicity we don't use KEK and KDK here.
type ContentKeyNode struct {
	schema.BaseNode

	leaf *schema.LeafNode
}

func (n *ContentKeyNode) Init(parent schema.NTNode, edge enc.ComponentPattern) {
	n.BaseNode.Init(parent, edge)

	pat, _ := enc.NamePatternFromStr("/<ckid>")
	n.leaf = &schema.LeafNode{}
	n.PutNode(pat, n.leaf)

	// The following setting is simply a default. One may overwrite it by policies after construction of the schema tree.
	n.leaf.Set(schema.PropCanBePrefix, false)
	n.leaf.Set(schema.PropMustBeFresh, false)
	n.leaf.Set(schema.PropFreshness, 876000*time.Hour)
	n.leaf.Set(schema.PropValidDuration, 876000*time.Hour)
	n.leaf.Set(schema.PropDataSigner, sec.NewSha256Signer())
	passAllChecker := func(enc.Matching, enc.Name, ndn.Signature, enc.Wire, schema.Context) schema.ValidRes {
		return schema.VrPass
	}
	schema.AddEventListener(n.leaf, schema.PropOnValidateData, passAllChecker)

	n.Self = n
}

func (n *ContentKeyNode) GenKey(matching enc.Matching) []byte {
	keybits := make([]byte, 32)
	rand.Read(keybits) // Should always succeed
	ckid := make([]byte, 8)
	rand.Read(ckid)
	matching["ckid"] = ckid
	// Produce the key. Storage policies will decide where to store the key
	n.leaf.Provide(matching, nil, enc.Wire{keybits}, schema.Context{})
	return ckid
}

func (n *ContentKeyNode) Encrypt(matching enc.Matching, ckid []byte, content enc.Wire) (enc.Wire, error) {
	if len(ckid) != 8 {
		return nil, fmt.Errorf("invalid content key id: %v", hex.EncodeToString(ckid))
	}
	matching["ckid"] = ckid
	res, keyWire := (<-n.leaf.Need(matching, nil, nil, schema.Context{schema.CkSupressInt: true})).Get()
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
	res, keyWire := (<-n.leaf.Need(matching, nil, nil, schema.Context{})).Get()
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

// GroupSigNode represents a subtree that supports group signature on a segmented node.
// TODO: Is it a better idea to let the user specify what `seg` is, instead of using fixed LeafNode?
// That may be way more complicated, and I'm not sure about the use case. (#BLACKBOX)
type GroupSigNode struct {
	schema.BaseNode

	seg  *schema.LeafNode
	meta *schema.LeafNode

	// nRoutines int
	// Segmentation threshold
	threshold int
}

func (n *GroupSigNode) Init(parent schema.NTNode, edge enc.ComponentPattern) {
	n.BaseNode.Init(parent, edge)

	// Segment packet
	pat, _ := enc.NamePatternFromStr("/seg/<8=seghash>")
	n.seg = &schema.LeafNode{}
	n.PutNode(pat, n.seg)
	n.seg.Set(schema.PropCanBePrefix, false)
	n.seg.Set(schema.PropMustBeFresh, false)
	n.seg.Set(schema.PropFreshness, 876000*time.Hour)
	n.seg.Set(schema.PropValidDuration, 876000*time.Hour)
	n.seg.Set(schema.PropDataSigner, sec.NewSha256Signer())
	passAllChecker := func(
		matching enc.Matching, _ enc.Name, sig ndn.Signature, covered enc.Wire, context schema.Context,
	) schema.ValidRes {
		if sig.SigType() != ndn.SignatureDigestSha256 {
			return schema.VrFail
		}
		seghash, ok := matching["seghash"].([]byte)
		if !ok || seghash == nil {
			return schema.VrFail
		}
		content, ok := context[schema.CkContent].(enc.Wire)
		if !ok {
			return schema.VrFail
		}
		h := sha256.New()
		for _, buf := range covered {
			_, err := h.Write(buf)
			if err != nil {
				return schema.VrFail
			}
		}
		if !bytes.Equal(h.Sum(nil), sig.SigValue()) {
			return schema.VrFail
		}
		// The name hash is the hash of content, not the signature covered part.
		h = sha256.New()
		for _, buf := range content {
			_, err := h.Write(buf)
			if err != nil {
				return schema.VrFail
			}
		}
		if bytes.Equal(h.Sum(nil), seghash) {
			return schema.VrBypass // Since segments are protected by the group signature, by pass the validation
		} else {
			return schema.VrFail
		}
	}
	schema.AddEventListener(n.seg, schema.PropOnValidateData, passAllChecker)

	pat, _ = enc.NamePatternFromStr("/32=meta")
	n.meta = &schema.LeafNode{} // This demo is not RDR and we don't handle version discovery
	n.PutNode(pat, n.meta)
	n.meta.Set(schema.PropCanBePrefix, false)
	n.meta.Set(schema.PropMustBeFresh, true)
	n.meta.Set(schema.PropFreshness, 10*time.Second)
	n.meta.Set(schema.PropValidDuration, 876000*time.Hour)
	// The signer and validator is set by the user

	n.threshold = 5000

	n.Self = n
}

func (n *GroupSigNode) Need(matching enc.Matching, context schema.Context) chan schema.NeedResult {
	retCh := make(chan schema.NeedResult, 1)
	go func() {
		// First obtain the metadata
		intRet, metaWire := (<-n.meta.Need(matching, nil, nil, context)).Get()
		if intRet != ndn.InterestResultData {
			n.Log.WithField("name", n.Apply(matching)).Warnf("Unable to fetch metadata: %v", intRet)
			retCh <- schema.NeedResult{Status: intRet, Content: nil}
			close(retCh)
			return
		}
		metaData := metaWire.Join()
		if len(metaData)%32 != 0 {
			n.Log.WithField("name", n.Apply(matching)).Warnf("The metadata is invalid")
			retCh <- schema.NeedResult{Status: ndn.InterestResultNone, Content: nil}
			close(retCh)
			return
		}

		// Fetch data segments
		nSegs := len(metaData) / 32
		ret := make(enc.Wire, 0, nSegs)
		for i := 0; i < nSegs; i++ {
			segHash := metaData[i*32 : i*32+32]
			matching["seghash"] = segHash
			// Here we don't use original context because:
			// 1. To avoid pollute from n.meta.Need
			// 2. Users have no need to override the setting for hashed segments
			// 3. To avoid race hazard
			intRet, segData := (<-n.seg.Need(matching, nil, nil, schema.Context{})).Get()
			if intRet != ndn.InterestResultData {
				n.Log.WithField("name", n.seg.Apply(matching)).Warnf("Failed to fetch segment")
				retCh <- schema.NeedResult{Status: intRet, Content: nil}
				close(retCh)
				return
			}
			ret = append(ret, segData...)
		}
		retCh <- schema.NeedResult{Status: ndn.InterestResultData, Content: ret}
		close(retCh)
	}()
	return retCh
}

func (n *GroupSigNode) Provide(matching enc.Matching, content enc.Wire, context schema.Context) {
	// Segmentation
	data := content.Join()
	nSegs := (len(data) + n.threshold - 1) / n.threshold
	metaWire := make(enc.Wire, nSegs)
	for i := 0; i < nSegs; i++ {
		st := i * n.threshold
		ed := utils.Min(i*n.threshold+n.threshold, len(data))
		h := sha256.New()
		h.Write(data[st:ed])
		segHash := h.Sum(nil)
		matching["seghash"] = segHash
		dataWire := n.seg.Provide(matching, nil, enc.Wire{data[st:ed]}, schema.Context{})
		if dataWire == nil {
			n.Log.WithField("name", n.seg.Apply(matching)).Warnf("Failed to provide segment data")
			return
		}
		metaWire[i] = segHash
	}
	// Provide metadata
	delete(matching, "seghash")
	dataWire := n.meta.Provide(matching, nil, metaWire, context)
	if dataWire == nil {
		n.Log.WithField("name", n.meta.Apply(matching)).Warnf("Failed to provide metadata")
		return
	}
}

// Get a property or callback event
func (n *GroupSigNode) Get(propName schema.PropKey) any {
	if ret := n.BaseNode.Get(propName); ret != nil {
		return ret
	}
	switch propName {
	case "Threshold":
		return n.threshold
	}
	return nil
}

// Set a property. Use Get() to update callback events.
func (n *GroupSigNode) Set(propName schema.PropKey, value any) error {
	if ret := n.BaseNode.Set(propName, value); ret == nil {
		return ret
	}
	switch propName {
	case "Threshold":
		return schema.PropertySet(&n.threshold, propName, value)
	}
	return ndn.ErrNotSupported{Item: string(propName)}
}

func (n *GroupSigNode) OnAttach(path enc.NamePattern, engine ndn.Engine) error {
	err := n.BaseNode.OnAttach(path, engine)
	if err != nil {
		return err
	}
	// Recover the data signer to SHA256
	// There are also other ways to do it, such as using Context
	n.seg.Set(schema.PropDataSigner, sec.NewSha256Signer())
	return nil
}
