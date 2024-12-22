package priority_queue

import (
	"container/heap"

	"golang.org/x/exp/constraints"
)

type Item[V any, P constraints.Ordered] struct {
	object   V
	priority P
	index    int
}

type wrapper[V any, P constraints.Ordered] []*Item[V, P]

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

func (pq *wrapper[V, P]) Push(x any) {
	item := x.(*Item[V, P])
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *wrapper[V, P]) Pop() any {
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
	return pq.pq.Len()
}

// Push pushes the 'value' onto the priority queue.
func (pq *Queue[V, P]) Push(value V, priority P) *Item[V, P] {
	ret := &Item[V, P]{
		object:   value,
		priority: priority,
	}
	heap.Push(&pq.pq, ret)
	return ret
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
	return heap.Pop(&pq.pq).(*Item[V, P]).object
}

// Update modifies the priority and value of the item
func (pq *Queue[V, P]) Update(item *Item[V, P], value V, priority P) {
	item.object = value
	item.priority = priority
	heap.Fix(&pq.pq, item.index)
}

// New creates a new priority queue. Not required to call.
func New[V any, P constraints.Ordered]() Queue[V, P] {
	return Queue[V, P]{wrapper[V, P]{}}
}
