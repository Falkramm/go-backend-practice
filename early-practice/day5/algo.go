package day5

func IsValid(s string) bool {
	var stack []rune
	var closeMap map[rune]rune = map[rune]rune{
		')': '(',
		']': '[',
		'}': '{',
	}
	var openSet map[rune]struct{} = map[rune]struct{}{
		'(': {},
		'[': {},
		'{': {},
	}
	for _, c := range s {
		if open, ok := closeMap[c]; ok {
			if len(stack) != 0 && stack[len(stack)-1] == open {
				stack = stack[:len(stack)-1]
			} else {
				return false
			}
			continue
		}
		if _, ok := openSet[c]; ok {
			stack = append(stack, c)
		}
	}
	return len(stack) == 0
}
