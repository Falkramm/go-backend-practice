package c01

// Alternate должна вернуть последовательность 1…n.
// При n <= 0 результат пустой, nil допустим.
// Обе горутины должны завершиться до возврата из функции.

import (
	"sync"
)

func Alternate(n int) []int {
	if n <= 0 {
		return []int{}
	}

	ch1, ch2 := make(chan int, 1), make(chan int, 1)
	var wg sync.WaitGroup
	res := make([]int, 0, n)
	f := func(in <-chan int, out chan<- int) {
		defer wg.Done()
		for v := range in {
			res = append(res, v)
			if v == n {
				close(out)
				return
			}

			out <- v + 1

			if v == n-1 {
				close(out)
				return
			}
		}
	}
	ch1 <- 1
	wg.Add(2)
	go f(ch1, ch2)
	go f(ch2, ch1)

	wg.Wait()
	return res
}
