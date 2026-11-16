// День 3 — практика: пощупать планировщик руками.
//
// Запуск:  go run . -exp 1   (2, 3)
//
// В exp2 есть TODO — сначала допиши его.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	// "sync"
	"syscall"
	"time"
)

func main() {
	exp := flag.Int("exp", 0, "номер эксперимента (1-3)")
	flag.Parse()
	switch *exp {
	case 1:
		exp1Preemption()
	case 2:
		exp2Parallelism()
	case 3:
		exp3Threads()
	default:
		fmt.Println("укажи -exp 1|2|3")
	}
}

// --- Эксперимент 1: асинхронное вытеснение (Go 1.14+) ---
//
// GOMAXPROCS=1 + вечный цикл без вызовов функций.
// На Go 1.13 эта программа зависла бы навсегда.
// Запусти и убедись, что sysmon вытесняет цикл и main доживает до печати.
func exp1Preemption() {
	runtime.GOMAXPROCS(1)
	go func() {
		for {
		}
	}()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("done: пустой цикл вытеснен, планировщик жив (спасибо, sysmon)")
}

// --- Эксперимент 2: GOMAXPROCS и CPU-bound работа ---
//
// Просуммируй числа от 0 до 4e9, раскидав работу на N горутин.
// Замеряем время при GOMAXPROCS=1 и GOMAXPROCS=NumCPU.
//
// TODO: реализуй parallelSum — подели диапазон на chunks кусков,
// каждый кусок суммируй в своей горутине, дождись всех через
// sync.WaitGroup и сложи результаты. Гонок быть не должно:
// каждая горутина пишет в СВОЮ ячейку слайса результатов.
func parallelSum(n uint64, chunks int) uint64 {
	ch := make(chan uint64)
	var sum uint64
	step := n / uint64(chunks)
	for i := range uint64(chunks) {
		l, r := i*step, (i+1)*step
		if i == uint64(chunks)-1 {
			r = n // последний кусок забирает остаток от деления
		}
		go func(l, r uint64) {
			sum := uint64(0)
			for i := l; i < r; i++ {
				sum += uint64(i)
			}
			ch <- sum
		}(l, r)
	}
	for range chunks {
		sum += <-ch
	}
	return sum
}

func exp2Parallelism() {
	const n = 4_000_000_000
	cpus := runtime.NumCPU()

	for _, p := range []int{1, cpus} {
		runtime.GOMAXPROCS(p)
		start := time.Now()
		sum := parallelSum(n, cpus)
		fmt.Printf("GOMAXPROCS=%-2d  sum=%d  time=%v\n", p, sum, time.Since(start))
	}
	fmt.Println("вопрос: почему ускорение не ровно в NumCPU раз?")
}

// --- Эксперимент 3: сисколлы рождают потоки, netpoller — нет ---
//
// 50 горутин виснут в СЫРОМ блокирующем сисколле read() на пайпе
// (мимо netpoller). Смотрим, сколько потоков создал рантайм.
func exp3Threads() {
	fmt.Printf("потоков создано на старте: %d\n", threadCount())

	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		fmt.Println("pipe:", err)
		os.Exit(1)
	}

	for i := 0; i < 50; i++ {
		go func() {
			buf := make([]byte, 1)
			syscall.Read(fds[0], buf) // сырой сисколл: поток блокируется в ядре
		}()
	}
	time.Sleep(300 * time.Millisecond)
	fmt.Printf("после 50 горутин в блокирующих сисколлах: %d потоков\n", threadCount())
	fmt.Println("сравни: 50 горутин в time.Sleep/сетевом ожидании потоков бы не добавили")
	fmt.Printf("(GOMAXPROCS=%d — а потоков сильно больше: вот он, handoff)\n", runtime.GOMAXPROCS(0))
}

func threadCount() int {
	return pprof.Lookup("threadcreate").Count()
}
