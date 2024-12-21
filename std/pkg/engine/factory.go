package engine

import (
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/engine/face"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	sec "github.com/zjkmxy/go-ndn/pkg/security"
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
