package laststoneweight

import "container/heap"

type maxHeap []int

func (h maxHeap) Len() int {
	return len(h)
}

func (h maxHeap) Less(i, j int) bool {
	return h[i] > h[j]
}

func (h maxHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *maxHeap) Push(x any) {
	*h = append(*h, x.(int))
}

func (h *maxHeap) Pop() any {
	res := (*h)[len(*h)-1]
	*h = (*h)[:len(*h)-1]
	return res
}

func lastStoneWeight(stones []int) int {
	h := maxHeap(stones)
	heap.Init(&h)
	for len(h) > 1 {
		x := heap.Pop(&h).(int) - heap.Pop(&h).(int)

		if x > 0 {
			heap.Push(&h, x)
		}
	}

	if len(h) == 0 {
		return 0
	}
	return h[0]
}
