package dummy

import (
	"errors"
	"sync"
	"time"
)

type event struct {
	t time.Time
	f func()
}

type Timer struct {
	now    time.Time
	events []event
	// Lock is not an very important thing because:
	//   1. Basic engine itself is single-threaded
	//   2. This timer is for test only, and there is a low chance for race.
	lock sync.Mutex
}

func NewTimer() *Timer {
	now, err := time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	if err != nil {
		return nil
	}
	return &Timer{
		now:    now,
		events: make([]event, 0),
	}
}

func (tm *Timer) Now() time.Time {
	return tm.now
}

func (tm *Timer) MoveForward(d time.Duration) {
	events := func() []event {
		tm.lock.Lock()
		defer tm.lock.Unlock()
		tm.now = tm.now.Add(d)
		ret := make([]event, len(tm.events))
		copy(ret, tm.events)
		return ret
	}()

	// Run events
	for i, e := range events {
		if e.f != nil {
			if e.t.Before(tm.now) {
				e.f()
				events[i].f = nil
			}
		}
	}

	func() {
		tm.lock.Lock()
		defer tm.lock.Unlock()
		tm.events = events
	}()
}

func (tm *Timer) Schedule(d time.Duration, f func()) func() error {
	t := tm.now.Add(d)
	tm.lock.Lock()
	defer tm.lock.Unlock()

	idx := len(tm.events)
	for i := range tm.events {
		if tm.events[i].f == nil {
			idx = i
			break
		}
	}
	if idx == len(tm.events) {
		tm.events = append(tm.events, event{
			t: t,
			f: f,
		})
	} else {
		tm.events[idx] = event{
			t: t,
			f: f,
		}
	}

	return func() error {
		if t.Before(tm.now) {
			return nil // Already past
		}
		if idx < len(tm.events) && tm.events[idx].t.Equal(t) && tm.events[idx].f != nil {
			tm.lock.Lock()
			defer tm.lock.Unlock()
			tm.events[idx].f = nil
			return nil
		} else {
			return errors.New("Event has already been canceled")
		}
	}
}

func (tm *Timer) Sleep(d time.Duration) {
	ch := make(chan struct{})
	tm.Schedule(d, func() {
		ch <- struct{}{}
		close(ch)
	})
	<-ch
}

func (_ *Timer) Nonce() []byte {
	return []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
}
