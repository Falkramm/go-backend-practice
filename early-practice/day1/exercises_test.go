package day1

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestInsertAt(t *testing.T) {
	// Запас capacity — ловушка: append без копии перезапишет s[2].
	s := make([]int, 3, 10)
	copy(s, []int{1, 2, 3})

	got := InsertAt(s, 1, 99)

	if !slices.Equal(got, []int{1, 99, 2, 3}) {
		t.Fatalf("InsertAt(s, 1, 99) = %v, ожидалось [1 99 2 3]", got)
	}
	if !slices.Equal(s, []int{1, 2, 3}) {
		t.Fatalf("исходный слайс изменился: %v — данные записались в общий массив", s)
	}

	// Вставка в начало и в конец.
	if got := InsertAt([]int{1, 2}, 0, 9); !slices.Equal(got, []int{9, 1, 2}) {
		t.Fatalf("вставка в начало: %v", got)
	}
	if got := InsertAt([]int{1, 2}, 2, 9); !slices.Equal(got, []int{1, 2, 9}) {
		t.Fatalf("вставка в конец: %v", got)
	}
}

func TestSafeTail(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}

	tail := SafeTail(s, 2)
	if !slices.Equal(tail, []int{4, 5}) {
		t.Fatalf("SafeTail = %v, ожидалось [4 5]", tail)
	}
	if cap(tail) != 2 {
		t.Fatalf("cap(tail) = %d, ожидалось 2: без ограничения cap чужой append залезет в s", cap(tail))
	}

	// Попытка испортить s через результат.
	_ = append(tail, 777)
	if !slices.Equal(s, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("append к результату испортил исходный слайс: %v", s)
	}

	// Данные при этом должны шариться (аллокаций не требуется).
	tail[0] = 40
	if s[3] != 40 {
		t.Fatal("SafeTail сделал копию — по условию нужно окно над той же памятью")
	}
	s[3] = 4
}

func TestFirstLine(t *testing.T) {
	huge := []byte("first line\nвторая строка\nи ещё")

	line := FirstLine(huge)
	if string(line) != "first line" {
		t.Fatalf("FirstLine = %q, ожидалось %q", line, "first line")
	}

	// Результат не должен смотреть в память huge.
	line[0] = 'X'
	if huge[0] != 'f' {
		t.Fatal("результат шарит память с huge — huge не сможет собраться GC")
	}
}

func TestReverseRunes(t *testing.T) {
	cases := map[string]string{
		"":      "",
		"a":     "a",
		"hello": "olleh",
		"жаба":  "абаж",
		"héllo": "olléh",
	}
	for in, want := range cases {
		if got := ReverseRunes(in); got != want {
			t.Fatalf("ReverseRunes(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

func TestConcat(t *testing.T) {
	parts := []string{"go", " ", "любит", " ", "билдер"}
	want := "go любит билдер"
	if got := Concat(parts); got != want {
		t.Fatalf("Concat = %q, ожидалось %q", got, want)
	}
	if got := Concat(nil); got != "" {
		t.Fatalf("Concat(nil) = %q, ожидалось пустая строка", got)
	}
}

func benchParts() []string {
	parts := make([]string, 2000)
	for i := range parts {
		parts[i] = fmt.Sprintf("part-%d;", i)
	}
	return parts
}

func BenchmarkConcatNaive(b *testing.B) {
	parts := benchParts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConcatNaive(parts)
	}
}

func BenchmarkConcat(b *testing.B) {
	parts := benchParts()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Concat(parts)
	}
}

var _ = strings.Builder{} // чтобы импорт strings был доступен сразу
