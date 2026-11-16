package dailytemperatures

func dailyTemperatures(temperatures []int) []int {
	stack := make([]int, 0, len(temperatures))
	res := make([]int, len(temperatures))

	for r := len(temperatures) - 1; r >= 0; r-- {
		for len(stack) > 0 && temperatures[stack[len(stack)-1]] <= temperatures[r] {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			res[r] = stack[len(stack)-1] - r
		}

		stack = append(stack, r)
	}
	return res
}
