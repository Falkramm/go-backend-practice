package combinationsum

import "slices"

func dp(candidates []int, buff []int, target int, res *[][]int) {
	if target == 0 {
		*res = append(*res, slices.Clone(buff))
		return
	}

	if len(candidates) == 0 {
		return
	}

	back := candidates[len(candidates)-1]
	num := 0
	dp(candidates[:len(candidates)-1], buff, target, res)
	for target >= back {
		buff = append(buff, back)
		target -= back
		dp(candidates[:len(candidates)-1], buff, target, res)
	}
	buff = buff[:len(buff)-num]

}
func combinationSum(candidates []int, target int) [][]int {
	buff := make([]int, 0, target)
	res := [][]int{}
	dp(candidates, buff, target, &res)
	return res
}
