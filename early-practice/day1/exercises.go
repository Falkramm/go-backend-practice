// День 1 — практика: слайсы, массивы, строки.
//
// Реализуй функции ниже вместо panic. Запуск тестов:
//
//	cd early-practice/day1
//	go test -v
//
// Бенчмарк конкатенации (когда всё зелёное):
//
//	go test -bench=. -benchmem
package day1

import (
	"bytes"
	"strings"
)

// InsertAt возвращает новый слайс с x, вставленным на позицию i.
// Исходный слайс s и его underlying-массив НЕ должны измениться —
// даже если у s есть запас capacity. Тесты это проверяют.
func InsertAt(s []int, i int, x int) []int {
	dst := make([]int, len(s))
	n := copy(dst, s)
	if n < i {
		panic("len is not enought.")
	}
	return append(dst[:i], append([]int{x}, dst[i:]...)...)
}

// SafeTail возвращает последние n элементов s так, чтобы вызывающий
// НЕ мог испортить данные s через append к результату.
// Аллокаций при этом быть не должно (подсказка: третий индекс).
func SafeTail(s []int, n int) []int {
	return s[len(s)-n : len(s) : len(s)]
}

// FirstLine возвращает первую строку huge (до '\n', без него).
// huge может быть огромным и коротко живущим: результат не должен
// удерживать его underlying-массив от сборки мусора.
func FirstLine(huge []byte) []byte {
	i := bytes.IndexByte(huge, '\n')
	if i == -1 {
		return nil
	}
	dst := make([]byte, len(huge[:i]))
	copy(dst, huge[:i])
	return dst
}

// ReverseRunes возвращает строку с символами в обратном порядке.
// Работает с любым UTF-8: "жаба" -> "абаж". Реверс по байтам сломает кириллицу.
func ReverseRunes(s string) string {
	r := []rune(s)

	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// Concat склеивает parts в одну строку через strings.Builder.
// (API Builder пиши по памяти — потом сверься с документацией.)
func Concat(parts []string) string {
	var bld strings.Builder
	for _, p := range parts {
		bld.WriteString(p)
	}
	return bld.String()
}

// ConcatNaive — для сравнения в бенчмарке, менять не нужно.
func ConcatNaive(parts []string) string {
	s := ""
	for _, p := range parts {
		s += p
	}
	return s
}
