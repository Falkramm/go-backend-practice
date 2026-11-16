package day7

import "testing"

func TestMaxSubarray(t *testing.T) {
	cases := []struct {
		in   []int
		want int
	}{
		{nil, 0},
		{[]int{5}, 5},
		{[]int{-2, 1, -3, 4, -1, 2, 1, -5, 4}, 6}, // [4 -1 2 1]
		{[]int{-3, -1, -2}, -1},                   // все отрицательные!
		{[]int{1, 2, 3}, 6},                       // весь массив
		{[]int{5, -9, 6, -2, 3}, 7},               // [6 -2 3]
		{[]int{0, -1, 0}, 0},
	}
	for _, c := range cases {
		if got := MaxSubarray(c.in); got != c.want {
			t.Errorf("MaxSubarray(%v) = %d, ожидалось %d", c.in, got, c.want)
		}
	}
}
