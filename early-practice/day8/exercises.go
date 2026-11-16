// День 8 — практика: паттерны конкурентности + context.
//
//	cd early-practice/day8
//	go mod tidy        ← один раз: подтянет errgroup
//	go test -race -v
package day8

import (
	"context"
	"golang.org/x/sync/errgroup"
	"sync"
	"sync/atomic"
	"time"
)

// 1. WorkerPool обрабатывает jobs пулом из n воркеров, возвращает
// СУММУ результатов process(job). Контекст отменили — прекратить
// раздачу задач и вернуть ctx.Err() (уже начатые задачи доработают).
// Каркас: канал задач + close + WaitGroup; сумму копить БЕЗ гонок
// (atomic или mutex — на выбор, -race проверит).
func WorkerPool(ctx context.Context, jobs []int, n int, process func(int) int) (int64, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var sum atomic.Int64
	var wg sync.WaitGroup
	var ch = make(chan int, n)
	for range n {
		wg.Add(1)
		go func(ctx context.Context, ch <-chan int) {
			ctx, cancel := context.WithCancel(ctx)
			defer wg.Done()
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-ch:
					if !ok {
						return
					}
					sum.Add(int64(process(v)))
				}
			}
		}(ctx, ch)
	}
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			return sum.Load(), ctx.Err()
		case ch <- job:
		}
	}
	close(ch)
	wg.Wait()
	return sum.Load(), nil
}

// 2. Merge (fan-in): слить НЕСКОЛЬКО каналов в один. Выходной канал
// закрывается, когда закрылись ВСЕ входные. Порядок значений любой.
// Классика: по горутине-перекачке на вход + WaitGroup + отдельная
// горутина, закрывающая out после Wait.
func Merge(chans ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)
	for _, ch := range chans {
		wg.Add(1)
		go func(ch <-chan int) {
			defer wg.Done()
			for {
				if v, ok := <-ch; ok {
					out <- v
				} else {
					break
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// 3. FirstResult: запустить fn ПАРАЛЛЕЛЬНО для каждого url и вернуть
// САМЫЙ БЫСТРЫЙ результат; остальные горутины не должны утечь
// (подумай про буфер канала — Д6!). Контекст отменили раньше — ctx.Err().
func FirstResult(ctx context.Context, urls []string, fn func(context.Context, string) string) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := make(chan string, len(urls))
	for _, url := range urls {
		go func(ctx context.Context, url string) {
			ch <- fn(ctx, url)
		}(ctx, url)
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case v := <-ch:
		cancel()
		return v, nil
	}
}

// 4. FetchAll: сходить за всеми urls ПАРАЛЛЕЛЬНО через errgroup.
// Любая ошибка -> вернуть её (остальные должны отмениться через ctx).
// Успех -> результаты В ПОРЯДКЕ urls (подсказка: пиши в results[i],
// это безопасно без мьютекса — почему?).
func FetchAll(ctx context.Context, urls []string, fetch func(context.Context, string) (string, error)) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]string, len(urls))
	for i, url := range urls {
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if res, err := fetch(ctx, url); err != nil {
				return err
			} else {
				results[i] = res
				return nil
			}
		})
	}

	err := g.Wait()
	return results, err
}

// 5. Debounce возвращает канал, в который уходит значение из in,
// только если после него ТИШИНА длиной d (новое значение перезапускает
// таймер — как поиск-подсказки: слать запрос, когда юзер ПЕРЕСТАЛ печатать).
// in закрылся -> дослать последнее (если было) и закрыть out.
// Подсказка: select по in и таймеру; time.NewTimer + Reset.
func Debounce(in <-chan int, d time.Duration) <-chan int {
	out := make(chan int)
	go func() {
		var lastValue int
		var hasValue bool
		timer := time.NewTimer(d)
		timer.Stop()
		for {
			select {
			case v, ok := <-in:
				if !ok {
					if hasValue {
						out <- lastValue
						hasValue = false
					}
					close(out)
					return
				}
				lastValue = v
				hasValue = true
				timer.Reset(d)
			case <-timer.C:
				if hasValue {
					out <- lastValue
					hasValue = false
				}
			}
		}
	}()

	return out
}
