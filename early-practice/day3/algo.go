package main

import "sort"

func MergeIntervals(in [][2]int) [][2]int {
	m := make(map[int]int)
	for _, interval := range in {
		m[interval[0]]++
		m[interval[1]]--
	}
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var sum int
	var res [][2]int
	var left int
	for _, k := range keys {
		if m[k] == 0 {
			continue
		}

		if m[k] < 0 && sum+m[k] == 0 {
			res = append(res, [2]int{left, k})
		}
		if m[k] > 0 && sum == 0 {
			left = k
		}
		sum += m[k]
	}
	return res
}
