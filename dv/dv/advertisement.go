package dv

import (
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

func (dv *DV) Advertise() {
}

func (dv *DV) onAdvertisementSyncInterest(
	interest ndn.Interest,
	rawInterest enc.Wire,
	sigCovered enc.Wire,
	reply ndn.ReplyFunc,
	deadline time.Time,
) {
	fmt.Println("Received Sync Interest")
}

// Global Interest handler
func (dv *DV) onAdvertisementInterest(
	interest ndn.Interest,
	rawInterest enc.Wire,
	sigCovered enc.Wire,
	reply ndn.ReplyFunc,
	deadline time.Time,
) {
	fmt.Println("Received Interest")
}
