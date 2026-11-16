package day1

import "testing"

func TestLongestUnique(t *testing.T) {
	cases := []struct {
		in      string
		wantLen int
	}{
		{"", 0},
		{"a", 1},
		{"bbbbb", 1},
		{"abcabcbb", 3}, // "abc"
		{"pwwkew", 3},   // "wke"
		{"abba", 2},     // "ab" или "ba"
		{"abcbdef", 5},  // "cbdef" — второй 'b' должен сдвинуть start на 2, не дальше
		{"abcdef", 6},   // без повторов — ответ находится ПОСЛЕ конца цикла
		{"жабажор", 5},  // "бажор" — руны, не байты
	}
	for _, c := range cases {
		got := LongestUnique(c.in)
		gotLen := len([]rune(got))
		if gotLen != c.wantLen {
			t.Errorf("LongestUnique(%q) = %q (длина %d), ожидалась длина %d",
				c.in, got, gotLen, c.wantLen)
		}
	}
}
