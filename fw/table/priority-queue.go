package table

import "container/heap"

// TODO: This PriorityQueue implementation has two problems:
// 1. Does not use generics (ref: https://go.dev/doc/tutorial/generics)
// 2. Have different type of receivers (ref: https://go.dev/tour/methods/8)

type PQItem struct {
	Object   interface{}
	Priority int64
	Index    int
}

type PriorityQueue []*PQItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PQItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Peek() *PQItem {
	return (*pq)[len(*pq)-1]
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *PQItem, value interface{}, priority int64) {
	item.Object = value
	item.Priority = priority
	heap.Fix(pq, item.Index)
}
