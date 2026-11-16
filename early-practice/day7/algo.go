package day7

func MaxSubarray(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	var res int = nums[0]
	var leftRangeSum int = nums[0]
	for _, num := range nums[1:] {
		if leftRangeSum < 0 {
			leftRangeSum = num
			res = max(res, num)
		} else {
			leftRangeSum += num
			res = max(res, leftRangeSum)
		}
	}
	return res
}
