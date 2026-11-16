package day6

func MergeSorted(a, b []int) []int {
	res := make([]int, 0, len(a)+len(b))
	for i, j := 0, 0; i < len(a) || j < len(b); {
		switch {
		case i == len(a):
			res = append(res, b[j:]...)
			j = len(b)
		case j == len(b):
			res = append(res, a[i:]...)
			i = len(a)
		case a[i] <= b[j]:
			res = append(res, a[i])
			i++
		case a[i] > b[j]:
			res = append(res, b[j])
			j++
		}
	}
	return res
}
