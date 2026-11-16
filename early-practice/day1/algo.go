package day1

func LongestUnique(s string) string {
	if len(s) == 0 {
		return ""
	}

	r := []rune(s)
	m := make(map[rune]int)
	start := 0
	r_l := 0
	r_r := 0
	for i, c := range r {
		if _, ok := m[c]; ok {
			if i-start > r_r-r_l+1 {
				r_l = start
				r_r = i - 1
			}
			start = max(start, m[c]+1)
		}
		m[c] = i
	}

	if len(r)-start > r_r-r_l+1 {
		r_l = start
		r_r = len(r) - 1
	}

	return string(r[r_l : r_r+1])
}
