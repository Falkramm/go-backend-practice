package mergeintervals

import "slices"

func merge(intervals [][]int) [][]int {
	if len(intervals) == 0 {
		return [][]int{}
	}

	intervalsS := slices.Clone(intervals)
	slices.SortFunc(
		intervalsS,
		func(a, b []int) int {
			return slices.Compare(a, b)
		},
	)

	res := [][]int{}
	l := intervalsS[0][0]
	r := intervalsS[0][1]

	for _, interval := range intervalsS {
		if interval[0] <= r {
			r = max(r, interval[1])
		} else {
			res = append(res, []int{l, r})
			l, r = interval[0], interval[1]
		}
	}

	res = append(res, []int{l, r})

	return res
}
