package day14

// 1. Map — применить функцию к каждому элементу
func Map[T comparable, U any](s []T, f func(T) U) []U {
	result := make([]U, 0, len(s))
	for _, v := range s {
		result = append(result, f(v))
	}
	return result
}

// 2. Keys — вернуть ключи мапы
func Keys[K comparable, V comparable](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}

// 3. Filter — оставить элементы, для которых pred == true
func Filter[T comparable](s []T, pred func(T) bool) []T {
	res := make([]T)
	for _, v := range s {
		if pred(v) {
			res = append(res, v)
		}
	}
	return res
}
// Напиши констрейнт и функцию Min, которая работает с int, float64, string и с пользовательскими типами на их основе:
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
