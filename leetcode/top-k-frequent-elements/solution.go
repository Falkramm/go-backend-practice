package topkfrequent

func topKFrequent(nums []int, k int) []int {
	mp := make(map[int]int)

	for _, v := range nums {
		mp[v]++
	}

	imp := make([][]int, len(nums)+1)
	for key, v := range mp {
		imp[v] = append(imp[v], key)
	}

	res := make([]int, 0, k)
	for i := len(imp) - 1; i >= 0 && k > 0; i-- {
		k -= len(imp[i])
		res = append(res, imp[i]...)
	}
	return res
}
