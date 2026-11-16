// Упражнение 2: охота на утечку. В этом сервисе ДВЕ утечки — найди их
// через pprof, НЕ читая код обработчиков (честно!), потом почини.
//
// Шаг 1 — запусти сервис:
//
//	cd early-practice/day4/leaky
//	go run .
//
// Шаг 2 — в другой вкладке дай нагрузку (200 запросов):
//
//	for i in $(seq 200); do curl -s localhost:8080/work > /dev/null; done
//
// Шаг 3 — смотри метрики и профили:
//
//	curl -s localhost:8080/stats                     ← горутины и куча
//	go tool pprof -top 'localhost:8080/debug/pprof/goroutine'
//	go tool pprof -top -inuse_space 'localhost:8080/debug/pprof/heap'
//
// Вопросы: сколько горутин? где они висят (какая строка)? какой
// аллокационный сайт держит память? Потом чини обе утечки и повторяй
// шаги 2-3 — горутины должны перестать расти.
package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	_ "net/http/pprof" // регистрирует /debug/pprof/* на DefaultServeMux
	"runtime"
	"sync"
)

// "кэш ответов" — второй подозреваемый
var (
	mu    sync.Mutex
	cache = map[string][]byte{}
)

func work(w http.ResponseWriter, r *http.Request) {
	results := make(chan string) // небуферизованный

	go func() {
		payload := make([]byte, 1<<20) // 1 МБ "результата"
		rand.Read(payload)

		mu.Lock()
		key := r.RemoteAddr + r.URL.RawQuery + fmt.Sprint(len(cache))
		cache[key] = payload
		mu.Unlock()

		results <- key
		results <- "extra" // штрих к портрету
	}()

	key := <-results
	mu.Lock()
	delete(cache, key)
	mu.Unlock()
	fmt.Fprintln(w, key)
	fmt.Fprintln(w, <-results)
}

func stats(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "goroutines: %d\nheap_inuse: %d MB\n",
		runtime.NumGoroutine(), m.HeapInuse>>20)
}

func main() {
	http.HandleFunc("/work", work)
	http.HandleFunc("/stats", stats)
	fmt.Println("listening on :8080 (pprof: /debug/pprof/)")
	http.ListenAndServe(":8080", nil)
}
