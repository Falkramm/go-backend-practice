package day6

import (
	"runtime"
	"slices"
	"testing"
	"time"
)

func TestGenerator(t *testing.T) {
	got := []int{}
	for v := range Generator(5) { // range выйдет, только если канал ЗАКРЫТ
		got = append(got, v)
	}
	if !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("Generator(5) отдал %v", got)
	}
	if vals := Drain(Generator(0)); len(vals) != 0 {
		t.Fatalf("Generator(0) должен закрыться пустым, got %v", vals)
	}
}

func TestDrain(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	close(ch)
	if got := Drain(ch); !slices.Equal(got, []int{10, 20}) {
		t.Fatalf("Drain = %v, ожидалось [10 20] — буфер закрытого канала дочитывается!", got)
	}
}

func TestTryRecv(t *testing.T) {
	ch := make(chan int, 1)

	if v, ok := TryRecv(ch); ok {
		t.Fatalf("пустой канал: TryRecv = (%d, true), ожидалось (0, false) БЕЗ блокировки", v)
	}
	ch <- 42
	if v, ok := TryRecv(ch); !ok || v != 42 {
		t.Fatalf("TryRecv = (%d, %v), ожидалось (42, true)", v, ok)
	}
}

func TestFetchWithTimeout(t *testing.T) {
	// быстрый ответ — успех
	res, ok := FetchWithTimeout(func() string { return "data" }, 100*time.Millisecond)
	if !ok || res != "data" {
		t.Fatalf("быстрый slow(): (%q, %v), ожидалось (data, true)", res, ok)
	}

	// медленный — таймаут
	res, ok = FetchWithTimeout(func() string {
		time.Sleep(200 * time.Millisecond)
		return "late"
	}, 20*time.Millisecond)
	if ok {
		t.Fatalf("ожидался таймаут, получено (%q, %v)", res, ok)
	}
}

func TestFetchWithTimeoutNoLeak(t *testing.T) {
	base := runtime.NumGoroutine()

	for i := 0; i < 50; i++ { // 50 таймаутов подряд
		FetchWithTimeout(func() string {
			time.Sleep(50 * time.Millisecond)
			return "late"
		}, time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond) // даём отставшим горутинам дозавершиться
	leaked := runtime.NumGoroutine() - base
	if leaked > 2 { // небольшой люфт на служебные
		t.Fatalf("утекло ~%d горутин после 50 таймаутов — канал результата без буфера?", leaked)
	}
}
