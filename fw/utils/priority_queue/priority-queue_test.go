package priority_queue_test

import (
	"testing"

	"github.com/pulsejet/ndnd/fw/utils/priority_queue"
	"github.com/stretchr/testify/assert"
)

func TestBasics(t *testing.T) {
	q := priority_queue.New[int, int]()
	assert.Equal(t, 0, q.Len())
	q.Push(1, 1)
	q.Push(2, 3)
	q.Push(3, 2)
	assert.Equal(t, 3, q.Len())
	assert.Equal(t, 1, q.PeekPriority())
	assert.Equal(t, 1, q.Pop())
	assert.Equal(t, 2, q.PeekPriority())
	assert.Equal(t, 3, q.Pop())
	assert.Equal(t, 2, q.Pop())
	assert.Equal(t, 0, q.Len())
}
