// Упражнение 1: предскажи, потом проверь.
//
// Для КАЖДОЙ помеченной строки запиши на бумажке: стек или куча?
// Потом проверь себя:
//
//	cd early-practice/day4/escape
//	go build -gcflags='-m' . 2>&1 | grep -v inline
//
// Разбор смотри по строкам вывода: "moved to heap", "escapes to heap",
// "does not escape".
package main

import "fmt"

type Point struct{ X, Y int }

var global *Point

func makePointVal() Point {
	p := Point{1, 2} // (1) ? stack
	return p
}

func makePointPtr() *Point {
	p := Point{3, 4} // (2) ? heap
	return &p
}

func saveGlobal() {
	p := Point{5, 6} // (3) ? heap
	global = &p
}

func localPtr() int {
	p := Point{7, 8} // (4) ? stack
	q := &p
	return q.X
}

func smallBuf() int {
	buf := make([]byte, 64) // (5) ? stack
	return len(buf)
}

func dynBuf(n int) int {
	buf := make([]byte, n) // (6) ? heap
	return len(buf)
}

func printed() {
	x := 42 // (7) ? heap because boxing
	fmt.Println(x)
}

func main() {
	_ = makePointVal()
	_ = makePointPtr()
	saveGlobal()
	_ = localPtr()
	_ = smallBuf()
	_ = dynBuf(10)
	printed()
}
