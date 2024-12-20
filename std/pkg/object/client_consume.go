package object

import (
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/log"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/schema/rdr"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

const maxObjectSeg = 1e9

// callback for consume API
// return true to continue fetching the object
type ConsumeCallback func(status *ConsumeState) bool

// arguments for the consume callback
type ConsumeState struct {
	// name of the object being consumed.
	name enc.Name
	// callback
	callback ConsumeCallback
	// error that occurred during fetching
	err error
	// raw data contents.
	content enc.Wire
	// fetching is completed
	complete bool
	// fetched metadata
	meta *rdr.MetaData
	// versioned object name
	fetchName enc.Name
	// fetching window
	wnd [3]int
	// from final block id
	segCnt int
}

// returns the name of the object being consumed
func (a *ConsumeState) Name() enc.Name {
	return a.name
}

// returns the error that occurred during fetching
func (a *ConsumeState) Error() error {
	return a.err
}

// returns true if the content has been completely fetched
func (a *ConsumeState) IsComplete() bool {
	return a.complete
}

// returns the currently available buffer in the content
// any subsequent calls to Content() will return data after the previous call
func (a *ConsumeState) Content() []byte {
	// return valid range of buffer (can be empty)
	buf := a.content[a.wnd[0]:a.wnd[1]].Join()

	// free buffers
	for i := a.wnd[0]; i < a.wnd[1]; i++ {
		a.content[i] = nil // gc
	}

	a.wnd[0] = a.wnd[1]
	return buf
}

// get the progress counter
func (a *ConsumeState) Progress() int {
	return a.wnd[1]
}

// get the max value for the progress counter (-1 for unknown)
func (a *ConsumeState) ProgressMax() int {
	return a.segCnt
}

// send a fatal error to the callback
func (a *ConsumeState) finalizeError(err error) {
	if !a.complete {
		a.err = err
		a.complete = true
		a.callback(a)
	}
}

func (c *Client) Consume(name enc.Name, callback ConsumeCallback) {
	c.consumeObject(&ConsumeState{
		name:      name,
		callback:  callback,
		err:       nil,
		content:   make(enc.Wire, 0), // just in case
		complete:  false,
		meta:      nil,
		fetchName: name,
		wnd:       [3]int{0, 0},
		segCnt:    -1,
	})
}

func (c *Client) consumeObject(state *ConsumeState) {
	name := state.fetchName

	// will segfault if name is empty
	if len(name) == 0 {
		state.finalizeError(fmt.Errorf("consume: name cannot be empty"))
		return
	}

	// fetch object metadata if the last name component is not a version
	if name[len(name)-1].Typ != enc.TypeVersionNameComponent {
		// when called with metadata, call with versioned name
		// state will always have the original object name
		if state.meta != nil {
			state.finalizeError(fmt.Errorf("consume: metadata does not have version component"))
			return
		}

		c.fetchMetadata(name, func(meta *rdr.MetaData, err error) {
			if err != nil {
				state.finalizeError(err)
				return
			}
			state.meta = meta
			state.fetchName = meta.Name
			c.consumeObject(state)
		})
		return
	}

	// passes ownership of state and callback to fetcher
	c.segfetch <- state
}

func (c *Client) fetchMetadata(
	name enc.Name,
	callback func(meta *rdr.MetaData, err error),
) {
	log.Debugf("consume: fetching object metadata %s", name)
	args := ExpressRArgs{
		Name: append(name,
			enc.NewStringComponent(enc.TypeKeywordNameComponent, "metadata"),
		),
		Config: &ndn.InterestConfig{
			CanBePrefix: true,
			MustBeFresh: true,
			Lifetime:    utils.IdPtr(time.Millisecond * 500),
		},
		Retries: 30,
	}
	c.ExpressR(args, func(args ndn.ExpressCallbackArgs) {
		if args.Result == ndn.InterestResultError {
			callback(nil, fmt.Errorf("consume: fetch failed with error %v", args.Error))
			return
		}

		if args.Result != ndn.InterestResultData {
			callback(nil, fmt.Errorf("consume: fetch failed with result %d", args.Result))
			return
		}

		// parse metadata
		metadata, err := rdr.ParseMetaData(enc.NewWireReader(args.Data.Content()), false)
		if err != nil {
			callback(nil, fmt.Errorf("consume: failed to parse object metadata %v", err))
			return
		}

		// clone fields for lifetime
		metadata.Name = metadata.Name.Clone()
		metadata.FinalBlockID = append([]byte{}, metadata.FinalBlockID...)
		callback(metadata, nil)
	})
}
