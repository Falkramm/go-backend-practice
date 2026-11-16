package groupanagrams

func groupAnagrams(strs []string) [][]string {
	mp := make(map[[26]int][]string)
	for _, str := range strs {
		alphs := [26]int{}
		for i := range len(str) {
			alphs[str[i]-'a']++
		}
		mp[alphs] = append(mp[alphs], str)
	}

	res := make([][]string, 0, len(mp))

	for _, v := range mp {
		res = append(res, v)
	}

	return res
}
