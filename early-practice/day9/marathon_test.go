package day9

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestM1_Alternate(t *testing.T) {
	var got []int
	for v := range Alternate(10) {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Fatalf("Alternate(10) = %v — порядок обязан быть строгим", got)
	}
	if got := Alternate(0); got == nil {
		t.Fatal("Alternate(0) должен вернуть закрытый канал, не nil")
	} else if _, ok := <-got; ok {
		t.Fatal("Alternate(0): канал должен быть пуст и закрыт")
	}
}

func TestM2_ParallelMap(t *testing.T) {
	var cur, max atomic.Int32
	f := func(ctx context.Context, x int) (int, error) {
		c := cur.Add(1)
		defer cur.Add(-1)
		for {
			m := max.Load()
			if c <= m || max.CompareAndSwap(m, c) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		return x * x, nil
	}

	in := []int{1, 2, 3, 4, 5, 6, 7, 8}
	got, err := ParallelMap(context.Background(), in, 3, f)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !slices.Equal(got, []int{1, 4, 9, 16, 25, 36, 49, 64}) {
		t.Fatalf("результаты/порядок: %v", got)
	}
	if m := max.Load(); m > 3 {
		t.Fatalf("одновременно работало %d, лимит 3", m)
	}
}

func TestM2_ParallelMapError(t *testing.T) {
	boom := errors.New("boom")
	f := func(ctx context.Context, x int) (int, error) {
		if x == 3 {
			return 0, boom
		}
		select {
		case <-time.After(2 * time.Second):
			return x, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	start := time.Now()
	_, err := ParallelMap(context.Background(), []int{1, 2, 3}, 3, f)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, ожидался boom", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("ошибка не отменила остальных")
	}
}

func TestM3_Stage(t *testing.T) {
	in := make(chan int, 3)
	in <- 1
	in <- 2
	in <- 3
	close(in)

	var got []int
	for v := range Stage(context.Background(), in, func(x int) int { return x * 10 }) {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{10, 20, 30}) {
		t.Fatalf("Stage = %v", got)
	}
}

func TestM3_StageCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan int) // никто никогда не напишет
	out := Stage(ctx, in, func(x int) int { return x })

	cancel()
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("после отмены не должно быть значений")
		}
		// закрылся — отлично
	case <-time.After(time.Second):
		t.Fatal("отменённая стадия не закрыла выход за 1с — горутина зависла?")
	}
}

func TestM4_Broadcast(t *testing.T) {
	in := make(chan int)
	outs := Broadcast(in, 3)
	if len(outs) != 3 {
		t.Fatalf("подписчиков %d, ожидалось 3", len(outs))
	}

	var wg sync.WaitGroup
	results := make([][]int, 3)
	for i, out := range outs {
		wg.Add(1)
		go func(i int, out <-chan int) {
			defer wg.Done()
			for v := range out {
				results[i] = append(results[i], v)
			}
		}(i, out)
	}

	for v := 1; v <= 5; v++ {
		in <- v
	}
	close(in)
	wg.Wait()

	for i, r := range results {
		if !slices.Equal(r, []int{1, 2, 3, 4, 5}) {
			t.Fatalf("подписчик %d получил %v, ожидалось [1..5] целиком", i, r)
		}
	}
}

func TestM5_RateLimit(t *testing.T) {
	in := make(chan int, 5)
	for v := 1; v <= 5; v++ {
		in <- v
	}
	close(in)

	start := time.Now()
	var got []int
	for v := range RateLimit(in, 100) { // 100/с → ~10мс на штуку
		got = append(got, v)
	}
	elapsed := time.Since(start)

	if !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("RateLimit = %v", got)
	}
	if elapsed < 35*time.Millisecond { // 5 значений => ≥ ~40мс (первое может уйти сразу)
		t.Fatalf("5 значений за %v — лимит не работает", elapsed)
	}
}

func TestM6_BatchBySize(t *testing.T) {
	in := make(chan int, 7)
	for v := 1; v <= 7; v++ {
		in <- v
	}
	close(in)

	var got [][]int
	for b := range Batch(in, 3, time.Hour) { // таймаут огромный — только размер
		got = append(got, b)
	}
	want := [][]int{{1, 2, 3}, {4, 5, 6}, {7}}
	if len(got) != 3 {
		t.Fatalf("пачек %d: %v", len(got), got)
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Fatalf("пачка %d = %v, ожидалось %v", i, got[i], want[i])
		}
	}
}

func TestM6_BatchByTimeout(t *testing.T) {
	in := make(chan int)
	out := Batch(in, 100, 50*time.Millisecond) // размер огромный — только таймаут

	in <- 1
	in <- 2

	select {
	case b := <-out:
		if !slices.Equal(b, []int{1, 2}) {
			t.Fatalf("пачка = %v, ожидалось [1 2]", b)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("пачка не пришла по таймауту")
	}
	close(in)
	if _, ok := <-out; ok {
		t.Fatal("после close(in) без остатка out должен закрыться")
	}
}
