// 🏗 Задача: Worker Pool
// Классика Go-интервью. Реализуй пул воркеров для обработки задач.

// Требования
// Базовые:

// Фиксированное число воркеров N обрабатывают задачи из общей очереди
// Клиент кладёт задачи через Submit(task)
// Нужно получать результаты (и ошибки) обработки
// Production-требования:

// Graceful shutdown — по сигналу перестать принимать новые задачи, дообработать взятые, корректно завершиться
// Backpressure — что делать, если задачи приходят быстрее, чем обрабатываются? Очередь не должна расти бесконечно
// Паника в воркере не должна убивать весь пул
// Отмена по контексту — ctx прокидывается в задачи
// Никаких утечек горутин после Shutdown()


package day15

import (
	"context"
	"sync"
	"fmt"
	"log"
)


type WorkerPool struct {
	tasks chan func()
	ctx context.Context
	wg sync.WaitGroup
	workerNum int
	shutdownOnce sync.Once
	// синхронизация
	// шатдаун сигнал
}

func worker(ctx context.Context, tasks <-chan func(), wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-tasks:
			if !ok {
				return
			}
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Println(err)
					}
				}()
				t()
			}()
		}
	}
}
func NewWorkerPool(n int) *WorkerPool {
	p := &WorkerPool{
		tasks : make(chan func(), n),
		ctx : context.Background(),
		workerNum : n,
	}

	for range p.workerNum {
		p.wg.Add(1)
		go worker(p.ctx, p.tasks, &p.wg)
	}

	return p
}

func (p *WorkerPool) Submit(task func()) (err error) {
	defer func() {
		if err := recover(); err != nil {
			err = fmt.Errorf("Submit is unavailable by %v", err)
		}
	}()
	p.tasks <- task
	return nil
}

func (p *WorkerPool) Shutdown() {
	p.shutdownOnce.Do(func() {
		close(p.tasks)
		p.wg.Wait()
	})
}
