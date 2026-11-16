package day2

func TwoSum(nums []int, target int) (int, int) {
	m := make(map[int]int)
	for i, v := range nums {
		if j, ok := m[target-v]; ok {
			return j, i
		}
		m[v] = i
	}
	return -1, -1
}
