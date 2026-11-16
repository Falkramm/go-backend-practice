package longestsubstring

func lengthOfLongestSubstring(s string) int {
	mp := make(map[byte]int)
	var res int
	var l int
	for r := range len(s) {
		if v, ok := mp[s[r]]; ok {
			l = max(l, v+1)
		}

		mp[s[r]] = r
		res = max(res, r-l+1)
	}
	return res
}
