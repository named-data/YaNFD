package dummy_test

import (
	"testing"
	"time"

	"github.com/named-data/ndnd/std/engine/dummy"
	"github.com/named-data/ndnd/std/utils"
	"github.com/stretchr/testify/require"
)

func TestClock(t *testing.T) {
	utils.SetTestingT(t)

	tm := dummy.NewTimer()
	require.Equal(t, utils.WithoutErr(time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")), tm.Now())
	tm.MoveForward(10 * time.Second)
	require.Equal(t, utils.WithoutErr(time.Parse(time.RFC3339, "1970-01-01T00:00:10Z")), tm.Now())
	tm.MoveForward(50 * time.Second)
	require.Equal(t, utils.WithoutErr(time.Parse(time.RFC3339, "1970-01-01T00:01:00Z")), tm.Now())
}

func TestSchedule(t *testing.T) {
	utils.SetTestingT(t)

	tm := dummy.NewTimer()
	val := 0
	tm.Schedule(10*time.Second, func() {
		val = 1
	})
	require.Equal(t, 0, val)
	tm.MoveForward(11 * time.Second)
	require.Equal(t, 1, val)
	lst := []int{0, 0, 0}
	tm.Schedule(10*time.Second, func() {
		lst[0] = 1
	})
	tm.Schedule(20*time.Second, func() {
		lst[1] = 2
	})
	tm.Schedule(15*time.Second, func() {
		lst[2] = 3
	})
	tm.MoveForward(11 * time.Second)
	require.Equal(t, []int{1, 0, 0}, lst)
	tm.MoveForward(5 * time.Second)
	require.Equal(t, []int{1, 0, 3}, lst)
	tm.MoveForward(5 * time.Second)
	require.Equal(t, []int{1, 2, 3}, lst)
}

func TestCancel(t *testing.T) {
	utils.SetTestingT(t)

	tm := dummy.NewTimer()
	val := 0
	cancel := tm.Schedule(10*time.Second, func() {
		val = 1
	})
	require.Equal(t, 0, val)
	cancel()
	tm.MoveForward(11 * time.Second)
	require.Equal(t, 0, val)

	lst := []int{0, 0, 0}
	tm.Schedule(10*time.Second, func() {
		lst[0] = 1
	})
	tm.Schedule(20*time.Second, func() {
		lst[1] = 2
	})
	cancel = tm.Schedule(15*time.Second, func() {
		lst[2] = 3
	})
	cancel()
	tm.MoveForward(21 * time.Second)
	require.Equal(t, []int{1, 2, 0}, lst)
}
