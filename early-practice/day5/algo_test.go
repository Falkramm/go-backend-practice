package day5

import "testing"

func TestIsValid(t *testing.T) {
	cases := map[string]bool{
		"":       true,
		"()":     true,
		"()[]{}": true,
		"([{}])": true,
		"(]":     false,
		"([)]":   false, // перекрёст — самый частый провал
		"(((":    false, // незакрытые
		")":      false, // закрытие без открытия
		"{[]}":   true,
		"]":      false,
	}
	for in, want := range cases {
		if got := IsValid(in); got != want {
			t.Errorf("IsValid(%q) = %v, ожидалось %v", in, got, want)
		}
	}
}
