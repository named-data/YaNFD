package dummy

import (
	"errors"
	"time"
)

type event struct {
	t time.Time
	f func()
}

type Timer struct {
	now    time.Time
	events []event
}

func NewTimer() *Timer {
	now, err := time.Parse(time.RFC3339, "1970-01-01T00:00:00Z00:00")
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

func (tm *Timer) Sleep(d time.Duration) {
	tm.now = tm.now.Add(d)
	// Run events
	newEvents := make([]event, 0, len(tm.events))
	for _, e := range tm.events {
		if e.f != nil {
			if !e.t.After(tm.now) {
				e.f()
			} else {
				newEvents = append(newEvents, e)
			}
		}
	}
	tm.events = newEvents
}

func (tm *Timer) Schedule(d time.Duration, f func()) func() error {
	t := tm.now.Add(d)
	tm.events = append(tm.events, event{
		t: t,
		f: f,
	})
	idx := len(tm.events) - 1
	return func() error {
		if idx < len(tm.events) && tm.events[idx].t.Equal(t) && tm.events[idx].f != nil {
			tm.events[idx].f = nil
			return nil
		} else {
			return errors.New("Event has already been canceled")
		}
	}
}

func (_ *Timer) Nonce() []byte {
	return []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
}
