package day6

import (
	"slices"
	"testing"
)

func TestMergeSorted(t *testing.T) {
	cases := []struct{ a, b, want []int }{
		{nil, nil, nil},
		{[]int{1}, nil, []int{1}},
		{nil, []int{2}, []int{2}},
		{[]int{1, 3, 5}, []int{2, 4}, []int{1, 2, 3, 4, 5}},
		{[]int{1, 2}, []int{3, 4}, []int{1, 2, 3, 4}},       // без перекрытия
		{[]int{3, 4}, []int{1, 2}, []int{1, 2, 3, 4}},       // b целиком раньше
		{[]int{1, 1, 1}, []int{1, 1}, []int{1, 1, 1, 1, 1}}, // дубликаты
		{[]int{-5, 0, 10}, []int{-10, 0, 20}, []int{-10, -5, 0, 0, 10, 20}},
	}
	for _, c := range cases {
		got := MergeSorted(c.a, c.b)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("MergeSorted(%v, %v) = %v, ожидалось %v", c.a, c.b, got, c.want)
		}
	}
}
