package priority_queue_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zjkmxy/go-ndn/pkg/utils/priority_queue"
)

func TestBasics(t *testing.T) {
	q := priority_queue.New[int, int]()
	assert.Equal(t, q.Len(), 0)
	q.Push(1, 1)
	q.Push(2, 3)
	q.Push(3, 2)
	assert.Equal(t, q.Len(), 3)
	assert.Equal(t, q.PeekPriority(), 1)
	assert.Equal(t, q.Pop(), 1)
	assert.Equal(t, q.PeekPriority(), 2)
	assert.Equal(t, q.Pop(), 3)
	assert.Equal(t, q.Pop(), 2)
	assert.Equal(t, q.Len(), 0)
}
