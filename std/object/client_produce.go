package object

import (
	"errors"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	rdr "github.com/named-data/ndnd/std/ndn/rdr_2024"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/utils"
)

// size of produced segment (~800B for header)
const pSegmentSize = 8000

type ProduceArgs struct {
	// name of the object to produce
	Name enc.Name
	// raw data contents
	Content enc.Wire
	// version of the object (defaults to unix timestamp, 0 for immutable)
	Version *uint64
	// time for which the object version can be cached (default 4s)
	FreshnessPeriod time.Duration
	// final expiry of the object (default 0 = no expiry)
	Expiry time.Time // TODO: not implemented
}

func (c *Client) Produce(args ProduceArgs) (enc.Name, error) {
	content := args.Content
	contentSize := 0
	for _, c := range content {
		contentSize += len(c)
	}
	if contentSize == 0 {
		return nil, errors.New("cannot produce empty object")
	}

	now := time.Now().UnixNano()
	if now < 0 { // > 1970
		return nil, errors.New("current unix time is negative")
	}

	version := uint64(now)
	if args.Version != nil {
		version = *args.Version
	}

	if args.FreshnessPeriod == 0 {
		args.FreshnessPeriod = 4 * time.Second
	}

	lastSeg := uint64((contentSize - 1) / pSegmentSize)
	finalBlockId := enc.NewSegmentComponent(lastSeg)

	cfg := &ndn.DataConfig{
		ContentType:  utils.IdPtr(ndn.ContentTypeBlob),
		Freshness:    utils.IdPtr(args.FreshnessPeriod),
		FinalBlockID: &finalBlockId,
	}

	// TODO: sign the data
	basename := append(args.Name, enc.NewVersionComponent(version))
	signer := sec.NewSha256Signer()

	// use a transaction to ensure the entire object is written
	c.store.Begin()
	defer c.store.Commit()

	var seg uint64
	for seg = 0; len(content) > 0; seg++ {
		name := append(basename, enc.NewSegmentComponent(seg))

		segContent := enc.Wire{}
		segContentSize := 0
		for len(content) > 0 && segContentSize < pSegmentSize {
			// append wire from content to segContent till segment is full
			sizeLeft := min(pSegmentSize-segContentSize, len(content[0]))
			newContent := content[0][:sizeLeft]
			segContent = append(segContent, newContent)
			segContentSize += len(newContent)

			// remove the content from the content slice
			content[0] = content[0][sizeLeft:]
			if len(content[0]) == 0 {
				content = content[1:]
			}
		}

		data, err := c.engine.Spec().MakeData(name, cfg, segContent, signer)
		if err != nil {
			return nil, err
		}

		err = c.store.Put(name, version, data.Wire.Join())
		if err != nil {
			return nil, err
		}
	}

	{ // write metadata packet
		name := append(args.Name,
			enc.NewStringComponent(enc.TypeKeywordNameComponent, "metadata"),
			enc.NewVersionComponent(version),
			enc.NewSegmentComponent(0),
		)
		content := rdr.MetaData{
			Name:         basename,
			FinalBlockID: finalBlockId.Bytes(),
		}

		data, err := c.engine.Spec().MakeData(name, cfg, content.Encode(), signer)
		if err != nil {
			return nil, err
		}

		err = c.store.Put(name, version, data.Wire.Join())
		if err != nil {
			return nil, err
		}
	}

	return basename, nil
}

func (c *Client) onInterest(args ndn.InterestHandlerArgs) {
	// TODO: consult security if we can send this
	wire, err := c.store.Get(args.Interest.Name(), args.Interest.CanBePrefix())
	if err != nil || wire == nil {
		return
	}
	args.Reply(enc.Wire{wire})
}
