package day8

// GroupAnagrams группирует слова-анаграммы.
// ["eat","tea","tan","ate","nat","bat"] -> [["eat","tea","ate"],["tan","nat"],["bat"]]
// Порядок групп и слов внутри группы не важен.

func getWordMap(str string) [256]byte {
	var wordMap [256]byte
	for _, c := range str {
		wordMap[c]++
	}
	return wordMap
}
func GroupAnagrams(words []string) [][]string {
	anagramsMap := make(map[[256]byte]int)
	groups := make([][]string, 0)
	for _, word := range words {
		wordMap := getWordMap(word)
		id, ok := anagramsMap[wordMap]
		if !ok {
			groups = append(groups, []string{word})
			anagramsMap[wordMap] = len(groups) - 1
			continue
		}

		groups[id] = append(groups[id], word)
	}
	return groups
}
