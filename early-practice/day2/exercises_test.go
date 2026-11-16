package day2

import (
	"errors"
	"maps"
	"sync"
	"testing"
)

func TestWordCount(t *testing.T) {
	got := WordCount("go любит go и ещё раз go")
	want := map[string]int{"go": 3, "любит": 1, "и": 1, "ещё": 1, "раз": 1}
	if !maps.Equal(got, want) {
		t.Fatalf("WordCount = %v, ожидалось %v", got, want)
	}
	empty := WordCount("")
	if empty == nil {
		t.Fatal("WordCount(\"\") вернул nil-мапу — а JSON из неё будет null, не {}")
	}
	if len(empty) != 0 {
		t.Fatalf("WordCount(\"\") = %v, ожидалась пустая мапа", empty)
	}
}

func TestTopWord(t *testing.T) {
	w, n := TopWord(map[string]int{"b": 2, "a": 2, "c": 1})
	if w != "a" || n != 2 {
		t.Fatalf("TopWord = (%q, %d), ожидалось (\"a\", 2) — при равенстве лексикографически меньшее", w, n)
	}
	if w, n := TopWord(nil); w != "" || n != 0 {
		t.Fatalf("TopWord(nil) = (%q, %d), ожидалось (\"\", 0)", w, n)
	}
}

func TestLookup(t *testing.T) {
	m := map[string]int{"zero": 0, "one": 1}

	v, err := Lookup(m, "zero")
	if err != nil || v != 0 {
		t.Fatalf("Lookup(zero) = (%d, %v) — ключ со значением 0 существует!", v, err)
	}
	_, err = Lookup(m, "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Lookup(nope): err = %v, ожидался ErrNotFound", err)
	}
}

func TestParseAllTypedNil(t *testing.T) {
	if err := ParseAll([]string{"ok", "bad", "ok"}); err == nil {
		t.Fatal("ParseAll с 'bad' должен вернуть ошибку")
	}
	err := ParseAll([]string{"ok", "ok"})
	if err != nil {
		t.Fatalf("ParseAll без 'bad' вернул %v (типа %T) — typed nil не починен", err, err)
	}
}

func TestCounterConcurrent(t *testing.T) {
	c := NewCounter()
	const goroutines = 50
	const perG = 200

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				c.Inc("hits")
				_ = c.Get("hits")
			}
		}()
	}
	wg.Wait()

	if got := c.Get("hits"); got != goroutines*perG {
		t.Fatalf("Get(hits) = %d, ожидалось %d — потерянные инкременты", got, goroutines*perG)
	}

	snap := c.Snapshot()
	snap["hits"] = -1
	if c.Get("hits") != goroutines*perG {
		t.Fatal("изменение снапшота повлияло на счётчик — Snapshot вернул внутреннюю мапу, а не копию")
	}
}
