package main

import (
	"slices"
	"testing"
)

func TestMergeIntervals(t *testing.T) {
	cases := []struct {
		in, want [][2]int
	}{
		{nil, nil},
		{[][2]int{{1, 4}}, [][2]int{{1, 4}}},
		{[][2]int{{1, 3}, {8, 10}, {2, 6}}, [][2]int{{1, 6}, {8, 10}}},
		{[][2]int{{1, 3}, {3, 5}}, [][2]int{{1, 5}}},           // касание — тоже слияние
		{[][2]int{{1, 10}, {2, 3}, {4, 5}}, [][2]int{{1, 10}}}, // вложенные
		{[][2]int{{5, 6}, {1, 2}}, [][2]int{{1, 2}, {5, 6}}},   // без пересечений, вход не отсортирован
		{[][2]int{{1, 2}, {2, 3}, {3, 4}}, [][2]int{{1, 4}}},   // цепочка
	}
	for _, c := range cases {
		got := MergeIntervals(slices.Clone(c.in))
		if !slices.Equal(got, c.want) {
			t.Errorf("MergeIntervals(%v) = %v, ожидалось %v", c.in, got, c.want)
		}
	}
}
