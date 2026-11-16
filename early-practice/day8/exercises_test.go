package day8

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"sort"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	jobs := make([]int, 100)
	for i := range jobs {
		jobs[i] = i + 1
	}
	sum, err := WorkerPool(context.Background(), jobs, 5, func(x int) int { return x * 2 })
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if sum != 10100 { // 2 * (1+...+100)
		t.Fatalf("sum = %d, ожидалось 10100", sum)
	}
}

func TestWorkerPoolCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу
	_, err := WorkerPool(ctx, []int{1, 2, 3}, 2, func(x int) int { return x })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, ожидался context.Canceled", err)
	}
}

func TestMerge(t *testing.T) {
	a := make(chan int, 3)
	b := make(chan int, 3)
	for _, v := range []int{1, 3, 5} {
		a <- v
	}
	for _, v := range []int{2, 4} {
		b <- v
	}
	close(a)
	close(b)

	var got []int
	for v := range Merge(a, b) { // range выйдет, только если out ЗАКРЫТ
		got = append(got, v)
	}
	sort.Ints(got)
	if !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("Merge собрал %v", got)
	}
}

func TestFirstResult(t *testing.T) {
	fn := func(ctx context.Context, url string) string {
		switch url {
		case "fast":
			time.Sleep(10 * time.Millisecond)
		default:
			time.Sleep(200 * time.Millisecond)
		}
		return url
	}
	base := runtime.NumGoroutine()

	res, err := FirstResult(context.Background(), []string{"slow1", "fast", "slow2"}, fn)
	if err != nil || res != "fast" {
		t.Fatalf("(%q, %v), ожидалось (fast, nil)", res, err)
	}

	time.Sleep(300 * time.Millisecond) // отставшие должны дозавершиться
	if leaked := runtime.NumGoroutine() - base; leaked > 2 {
		t.Fatalf("~%d горутин утекло — медленным некуда деть результат?", leaked)
	}
}

func TestFetchAll(t *testing.T) {
	fetch := func(ctx context.Context, url string) (string, error) {
		return "data:" + url, nil
	}
	got, err := FetchAll(context.Background(), []string{"a", "b", "c"}, fetch)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !slices.Equal(got, []string{"data:a", "data:b", "data:c"}) {
		t.Fatalf("результаты не в порядке urls: %v", got)
	}
}

func TestFetchAllError(t *testing.T) {
	boom := errors.New("boom")
	fetch := func(ctx context.Context, url string) (string, error) {
		if url == "bad" {
			return "", boom
		}
		select { // «медленный» url должен успеть увидеть отмену
		case <-time.After(2 * time.Second):
			return "data:" + url, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	start := time.Now()
	_, err := FetchAll(context.Background(), []string{"slow", "bad"}, fetch)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, ожидался boom", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("ошибка не отменила медленного соседа — errgroup.WithContext не использован?")
	}
}

func TestDebounce(t *testing.T) {
	in := make(chan int)
	out := Debounce(in, 50*time.Millisecond)

	go func() {
		in <- 1 // быстрая очередь: 1, 2, 3 почти подряд
		time.Sleep(10 * time.Millisecond)
		in <- 2
		time.Sleep(10 * time.Millisecond)
		in <- 3
		time.Sleep(100 * time.Millisecond) // тишина → 3 должен уйти
		in <- 4
		close(in) // закрытие → 4 досылается
	}()

	var got []int
	for v := range out {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{3, 4}) {
		t.Fatalf("Debounce отдал %v, ожидалось [3 4]", got)
	}
}
