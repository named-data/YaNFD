package engine

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/engine/face"
	"github.com/named-data/ndnd/std/ndn"
	sec "github.com/named-data/ndnd/std/security"
)

// TODO: this API will change once there is a real security model
func NewBasicEngine(face face.Face) ndn.Engine {
	timer := basic.NewTimer()
	cmdSigner := sec.NewSha256IntSigner(timer)
	cmdValidator := func(enc.Name, enc.Wire, ndn.Signature) bool {
		return true
	}
	return basic.NewEngine(face, timer, cmdSigner, cmdValidator)
}

func NewUnixFace(addr string) face.Face {
	return face.NewStreamFace("unix", addr, true)
}
