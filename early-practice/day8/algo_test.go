package day8

import (
	"sort"
	"testing"
)

func normalize(groups [][]string) [][]string {
	for _, g := range groups {
		sort.Strings(g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i][0] < groups[j][0]
	})
	return groups
}

func TestGroupAnagrams(t *testing.T) {
	got := normalize(GroupAnagrams([]string{"eat", "tea", "tan", "ate", "nat", "bat"}))
	want := [][]string{{"ate", "eat", "tea"}, {"bat"}, {"nat", "tan"}}

	if len(got) != len(want) {
		t.Fatalf("групп %d, ожидалось %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("группа %d: %v, ожидалось %v", i, got[i], want[i])
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Fatalf("группа %d: %v, ожидалось %v", i, got[i], want[i])
			}
		}
	}

	if g := GroupAnagrams(nil); len(g) != 0 {
		t.Fatalf("nil вход: %v", g)
	}
	if g := normalize(GroupAnagrams([]string{"a"})); len(g) != 1 || g[0][0] != "a" {
		t.Fatalf("одно слово: %v", g)
	}
}
