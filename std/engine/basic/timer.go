package basic

import (
	"crypto/rand"
	"errors"
	"time"

	"github.com/named-data/ndnd/std/ndn"
)

type Timer struct{}

func NewTimer() ndn.Timer {
	return Timer{}
}

func (Timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (Timer) Schedule(d time.Duration, f func()) func() error {
	t := time.AfterFunc(d, f)
	return func() error {
		if t != nil {
			t.Stop()
			t = nil
			return nil
		} else {
			return errors.New("event has already been canceled")
		}
	}
}

func (Timer) Now() time.Time {
	return time.Now()
}

func (Timer) Nonce() []byte {
	// After go1.20 rand.Seed does not need to be called manually.
	buf := make([]byte, 8)
	n, _ := rand.Read(buf) // Should always succeed
	return buf[:n]
}
