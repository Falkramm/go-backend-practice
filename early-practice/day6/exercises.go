// День 6 — практика: каналы и select.
//
//	cd early-practice/day6
//	go test -race -v      ← новый go, -race работает!
package day6

import (
	"time"
)

// 1. Generator возвращает канал, из которого можно прочитать числа
// 0..n-1, после чего канал закрывается. Писать должна отдельная
// горутина (вызов Generator не блокируется).
// Помни правило владения: закрывает тот, кто пишет.
func Generator(n int) <-chan int {
	ch := make(chan int)
	go func() {
		for i := 0; i < n; {
			select {
			case ch <- i:
				i++
			}
		}
		close(ch)
	}()
	return ch
}

// 2. Drain вычитывает из канала ВСЁ до его закрытия и возвращает слайсом.
func Drain(ch <-chan int) []int {
	var rs []int
	for v := range ch {
		rs = append(rs, v)
	}
	return rs
}

// 3. TryRecv — неблокирующий приём: если в канале есть значение —
// вернуть (значение, true), если пусто — сразу (0, false), не вися.
// (Зеркало твоего trySend из теории.)
func TryRecv(ch <-chan int) (int, bool) {
	select {
	case v := <-ch:
		return v, true
	default:
		return 0, false
	}
}

// 4. FetchWithTimeout запускает slow() в горутине и ждёт результат
// не дольше timeout. Успех -> (результат, true), таймаут -> ("", false).
//
// ГЛАВНОЕ ТРЕБОВАНИЕ: горутина со slow() НЕ должна утекать —
// даже когда случился таймаут, она обязана иметь возможность
// завершиться (тест считает горутины!). Вспомни ловушку 1 и её фикс.
func FetchWithTimeout(slow func() string, timeout time.Duration) (string, bool) {
	ch := make(chan string, 1)
	go func() {
		ch <- slow()
	}()

	select {
	case v := <-ch:
		return v, true
	case <-time.After(timeout):
		return "", false
	}
}
