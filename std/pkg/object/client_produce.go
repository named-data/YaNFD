package object

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

type ProduceArgs struct {
	// name of the object to produce.
	Name enc.Name
	// raw data contents.
	Content []byte
	// version of the object (defaults to unix timestamp).
	Version uint64
	// time for which the object version can be cached (default 4s).
	FreshnessPeriod time.Duration
	// final expiry of the object (default 0 = no expiry).
	Expiry time.Time
}

func (c *Client) Produce(args ProduceArgs) error {
	return nil
}
