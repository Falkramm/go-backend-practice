package subsets

func subsets(nums []int) [][]int {
	res := make([][]int, 1<<len(nums))
	for mask := range 1 << len(nums) {
		for i, v := range nums {
			if mask&(1<<i) != 0 {
				res[mask] = append(res[mask], v)
			}
		}
	}
	return res
}
