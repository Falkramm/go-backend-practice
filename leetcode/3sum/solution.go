package threesum

import "slices"

func threeSum(nums []int) [][]int {
	setRes := make(map[[3]int]struct{})
	for l := range len(nums) {
		mp := make(map[int]int)
		for r := l + 1; r < len(nums); r++ {
			if _, ok := mp[-nums[l]-nums[r]]; ok {
				key := [3]int{nums[l], nums[r], -nums[l] - nums[r]}
				slices.Sort(key[:])
				setRes[key] = struct{}{}
			}
			mp[nums[r]]++
		}
	}

	res := make([][]int, 0, len(setRes))
	for k, _ := range setRes {
		res = append(res, k[:])
	}
	return res
}
