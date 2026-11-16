package day2

import "testing"

func TestTwoSum(t *testing.T) {
	cases := []struct {
		nums   []int
		target int
		i, j   int
	}{
		{[]int{2, 7, 11, 15}, 9, 0, 1},
		{[]int{3, 2, 4}, 6, 1, 2},
		{[]int{3, 3}, 6, 0, 1},      // одинаковые элементы — разные индексы
		{[]int{5}, 10, -1, -1},      // один элемент сам с собой не пара
		{[]int{1, 2}, 10, -1, -1},   // нет пары
		{[]int{-3, 4, 90}, 1, 0, 1}, // отрицательные
		{nil, 0, -1, -1},
	}
	for _, c := range cases {
		i, j := TwoSum(c.nums, c.target)
		if i != c.i || j != c.j {
			t.Errorf("TwoSum(%v, %d) = (%d, %d), ожидалось (%d, %d)",
				c.nums, c.target, i, j, c.i, c.j)
		}
	}
}
