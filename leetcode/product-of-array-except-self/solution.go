package productexceptself

func productExceptSelf(nums []int) []int {
	res := make([]int, len(nums))
	var product int = 1
	for i := range len(res) {
		res[i] = product
		product *= nums[i]
	}

	product = 1
	for i := len(res) - 1; i >= 0; i-- {
		res[i] *= product
		product *= nums[i]
	}
	return res
}
