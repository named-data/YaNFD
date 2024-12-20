package object

import (
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

type Client struct {
	// underlying API engine.
	engine ndn.Engine
	// segment fetcher.
	fetcher *rrSegFetcher

	// stop the client
	stop chan bool
	// outgoing interest pipeline
	outpipe chan ExpressRArgs
	// incoming data pipeline for fetcher
	seginpipe chan rrSegHandleDataArgs
	// queue for segment fetcher
	segfetch chan *ConsumeState
	// check segment fetcher
	segcheck chan bool
}

func NewClient(engine ndn.Engine) *Client {
	client := new(Client)
	client.engine = engine
	client.fetcher = newRrSegFetcher(client)

	client.stop = make(chan bool)
	client.outpipe = make(chan ExpressRArgs, 1024)
	client.seginpipe = make(chan rrSegHandleDataArgs, 1024)
	client.segfetch = make(chan *ConsumeState, 128)
	client.segcheck = make(chan bool, 2)

	return client
}

func (c *Client) Start() error {
	go c.run()
	return nil
}

func (c *Client) Stop() {
	c.stop <- true
}

func (c *Client) Engine() ndn.Engine {
	return c.engine
}

func (c *Client) run() {
	for {
		select {
		case <-c.stop:
			return
		case args := <-c.outpipe:
			c.expressRImpl(args)
		case args := <-c.seginpipe:
			c.fetcher.handleData(args.args, args.state)
		case state := <-c.segfetch:
			c.fetcher.add(state)
		case <-c.segcheck:
			c.fetcher.doCheck()
		}
	}
}
