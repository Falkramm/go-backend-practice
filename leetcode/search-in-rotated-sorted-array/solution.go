package rotatedsearch

func getMaxPos(nums []int) int {
	l, r := 0, len(nums)-1
	for l+1 < r {
		m := (l + r) / 2
		if nums[m] > nums[0] {
			l = m
		} else {
			r = m - 1
		}
	}
	if nums[r] > nums[l] {
		return r
	}

	return l
}

func searchAsc(nums []int, target int) int {
	if len(nums) == 0 {
		return -1
	}
	l, r := 0, len(nums)-1
	for l < r {
		if nums[l] > target || nums[r] < target {
			return -1
		}

		m := (l + r) / 2
		switch {
		case nums[m] > target:
			r = m - 1
		case nums[m] < target:
			l = m + 1
		default:
			return m
		}
	}
	if nums[l] != target {
		return -1
	}
	return l
}
func search(nums []int, target int) int {
	maxPos := getMaxPos(nums)
	l := searchAsc(nums[:maxPos+1], target)
	if l != -1 {
		return l
	}

	r := searchAsc(nums[maxPos+1:], target)
	if r == -1 {
		return -1
	}

	return r + maxPos + 1
}
