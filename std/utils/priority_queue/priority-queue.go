package priority_queue

import (
	"github.com/pulsejet/ndnd/std/utils/heap"
	"golang.org/x/exp/constraints"
)

type item[V any, P constraints.Ordered] struct {
	object   V
	priority P
	index    int
}

type wrapper[V any, P constraints.Ordered] []*item[V, P]

// Queue represents a priority queue with MINIMUM priority.
type Queue[V any, P constraints.Ordered] struct {
	pq wrapper[V, P]
}

func (pq *wrapper[V, P]) Len() int {
	return len(*pq)
}

func (pq *wrapper[V, P]) Less(i, j int) bool {
	return (*pq)[i].priority < (*pq)[j].priority
}

func (pq *wrapper[V, P]) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].index = i
	(*pq)[j].index = j
}

func (pq *wrapper[V, P]) Push(x *item[V, P]) {
	item := x
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *wrapper[V, P]) Pop() *item[V, P] {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// Len returns the length of the priroity queue.
func (pq *Queue[V, P]) Len() int {
	return len(pq.pq)
}

// Push pushes the 'value' onto the priority queue.
func (pq *Queue[V, P]) Push(value V, priority P) int {
	ret := &item[V, P]{
		object:   value,
		priority: priority,
	}
	heap.Push[*item[V, P]](&pq.pq, ret)
	return ret.index
}

// Peek returns the minimum element of the priority queue without removing it.
func (pq *Queue[V, P]) Peek() V {
	return pq.pq[0].object
}

// Peek returns the minimum element's priority.
func (pq *Queue[V, P]) PeekPriority() P {
	return pq.pq[0].priority
}

// Pop removes and returns the minimum element of the priority queue.
func (pq *Queue[V, P]) Pop() V {
	return heap.Pop[*item[V, P]](&pq.pq).object
}

// Update modifies the priority and value of the item with 'index' in the queue.
// returns the updated index.
func (pq *Queue[V, P]) Update(index int, value V, priority P) int {
	if index < 0 || index >= len(pq.pq) {
		return -1
	}
	it := pq.pq[index]
	it.object = value
	it.priority = priority
	heap.Fix[*item[V, P]](&pq.pq, it.index)
	return it.index
}

// New creates a new priority queue. Not required to call.
func New[V any, P constraints.Ordered]() Queue[V, P] {
	return Queue[V, P]{wrapper[V, P]{}}
}
