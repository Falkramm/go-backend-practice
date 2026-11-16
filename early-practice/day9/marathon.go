// День 9 — live-coding марафон: 6 задач.
//
//	cd early-practice/day9
//	go get golang.org/x/sync/errgroup   ← понадобится для М2
//	go test -race -run TestM1 -v        ← гоняй по одной!
package day9

import (
	"context"
	"golang.org/x/sync/errgroup"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// ===== М1. Эстафета =====
// Alternate возвращает канал, из которого читаются 1, 2, 3, ..., n
// СТРОГО ПО ПОРЯДКУ — но нечётные числа обязана писать одна горутина,
// чётные — другая. Классика «синхронизуй две горутины».
// Подсказка: эстафетная палочка — пара небуферизованных каналов
// (или один туда-обратно).
func Alternate(n int) <-chan int {
	out := make(chan int)
	if n < 1 {
		close(out)
		return out
	}
	ch1, ch2 := make(chan int), make(chan int)

	f := func(ch1 <-chan int, ch2 chan<- int) {
		for {
			v, ok := <-ch1
			if !ok {
				return
			}
			out <- v
			if v == n {
				close(out)
				return
			}
			ch2 <- (v + 1)
			if v == n-1 {
				return
			}
		}
	}

	go f(ch1, ch2)
	go f(ch2, ch1)
	ch1 <- 1

	return out
}

// ===== М2. Параллельный map с лимитом =====
// ParallelMap применяет f к каждому элементу in ПАРАЛЛЕЛЬНО,
// но не более limit одновременно. Результаты — в порядке входа.
// Ошибка любой f -> вернуть её, остальных отменить (ctx).
func ParallelMap(ctx context.Context, in []int, limit int, f func(context.Context, int) (int, error)) ([]int, error) {
	out := make([]int, len(in))
	var lastTaken atomic.Int64
	g, ctx := errgroup.WithContext(ctx)
	for range min(limit, len(in)) {
		g.Go(
			func() error {
				for {
					if err := ctx.Err(); err != nil {
						return err
					}
					i := lastTaken.Add(1) - 1
					if i >= int64(len(in)) {
						return nil
					}
					v, err := f(ctx, in[i])
					if err != nil {
						return err
					}

					out[i] = v
				}
			})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return out, nil
}

// ===== М3. Стадия конвейера =====
// Stage читает из in, применяет f и пишет в выходной канал.
// Выход закрывается, когда in закрыт и дочитан ЛИБО ctx отменён.
// Ни при каком сценарии горутина стадии не должна зависнуть.
func Stage(ctx context.Context, in <-chan int, f func(int) int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- f(v):
				}
			}
		}
	}()

	return out
}

// ===== М4. Broadcast (размножитель) =====
// Broadcast раздаёт КАЖДОЕ значение из in ВСЕМ n подписчикам
// (каждый подписчик получает полную копию потока).
// Все выходные каналы закрываются после закрытия in.
func Broadcast(in <-chan int, n int) []<-chan int {
	out := make([]chan int, n)
	outRecv := make([]<-chan int, n)
	for i := range out {
		ch := make(chan int)
		out[i], outRecv[i] = ch, ch
	}
	go func() {
		defer func() {
			for _, ch := range out {
				close(ch)
			}
		}()
		for {
			v, ok := <-in
			if !ok {
				return
			}
			var wg sync.WaitGroup
			for _, ch := range out {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ch <- v
				}()
			}
			wg.Wait()
		}
	}()
	return outRecv
}

// ===== М5. Ограничитель скорости =====
// RateLimit пропускает значения из in не чаще rate штук в секунду.
// Значения и порядок сохраняются. in закрылся -> out закрывается.
// Подсказка: time.Ticker (и не забудь Stop — Д6, ловушка 2!).
func RateLimit(in <-chan int, rate int) <-chan int {
	out := make(chan int)
	go func() {
		interval := time.Second / time.Duration(rate)
		timer := time.NewTicker(interval)
		defer timer.Stop()
		defer close(out)
		for {
			v, ok := <-in
			if !ok {
				return
			}

			<-timer.C

			out <- v
		}
	}()
	return out
}

// ===== М6. Батчер =====
// Batch копит значения из in и отдаёт их ПАЧКАМИ в out:
// пачка уходит, когда набралось size элементов ИЛИ прошло timeout
// с момента ПЕРВОГО элемента в текущей пачке (что раньше).
// in закрылся -> дослать неполную пачку (если есть) и закрыть out.
// Это реальная прод-задача: батчинг записей в базу/кафку.
func Batch(in <-chan int, size int, timeout time.Duration) <-chan []int {
	out := make(chan []int)

	go func() {
		defer close(out)
		batch := make([]int, 0, size)
		timer := time.NewTicker(timeout)
		timer.Stop()
		for {
			select {
			case v, ok := <-in:
				if !ok {
					if len(batch) > 0 {
						out <- slices.Clone(batch)
						batch = batch[:0]
					}
					return
				}
				batch = append(batch, v)
				if len(batch) == size {
					out <- slices.Clone(batch)
					batch = batch[:0]
					timer.Stop()
				}

				if len(batch) == 1 {
					timer.Reset(timeout)
				}
			case <-timer.C:
				if len(batch) > 0 {
					out <- slices.Clone(batch)
					batch = batch[:0]
				}
				timer.Stop()
			}
		}
	}()

	return out
}
