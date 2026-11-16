package kthlargestarray

import "container/heap"
import "fmt"

type MaxHeap []int

func (h MaxHeap) Len() int {
	return len(h)
}

func (h MaxHeap) Less(i, j int) bool {
	return h[i] < h[j]
}

func (h MaxHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *MaxHeap) Push(x any) {
	*h = append(*h, x.(int))
}

func (h *MaxHeap) Pop() any {
	res := (*h)[len(*h)-1]
	*h = (*h)[:len(*h)-1]
	return res
}

func findKthLargest(nums []int, k int) int {
	h := MaxHeap(nums[:k])
	heap.Init(&h)
	fmt.Println(h)
	for _, num := range nums[k:] {
		if num > h[0] {
			heap.Pop(&h)
			heap.Push(&h, num)
		}
	}
	return h[0]
}
