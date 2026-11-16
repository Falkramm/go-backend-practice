package day7

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCacheConcurrent(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup

	// 10 писателей × 100 записей
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				c.Set(fmt.Sprintf("k%d", i%20), Item{Value: "v", Score: w})
			}
		}(w)
	}
	// 10 читателей × 100 чтений
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				c.Get(fmt.Sprintf("k%d", i%20))
			}
		}()
	}
	wg.Wait()

	if _, ok := c.Get("k5"); !ok {
		t.Fatal("k5 должен существовать")
	}
}

func TestCacheHitsAccurate(t *testing.T) {
	c := NewCache()
	c.Set("a", Item{Value: "x"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Get("a") // hit
			}
		}()
	}
	wg.Wait()

	if got := c.Hits(); got != 5000 {
		t.Fatalf("Hits = %d, ожидалось ровно 5000 — инкременты теряются", got)
	}
}

func TestCacheReturnsCopy(t *testing.T) {
	c := NewCache()
	c.Set("a", Item{Value: "orig", Score: 1})

	it, _ := c.Get("a")
	it.Value = "hacked" // мутируем ПОЛУЧЕННОЕ

	it2, _ := c.Get("a")
	if it2.Value != "orig" {
		t.Fatal("мутация результата Get изменила данные в кеше — наружу ушёл общий указатель")
	}
}

func TestGetConfigOnce(t *testing.T) {
	atomic.StoreInt64(&initCalls, 0)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cfg := GetConfig(); cfg == nil {
				t.Error("GetConfig вернул nil")
			}
		}()
	}
	wg.Wait()

	if n := atomic.LoadInt64(&initCalls); n != 1 {
		t.Fatalf("heavyInit вызван %d раз, ожидалось ровно 1", n)
	}
}

func TestCounters(t *testing.T) {
	for name, c := range map[string]interface {
		Inc()
		Load() int64
	}{"mutex": &MutexCounter{}, "atomic": &AtomicCounter{}} {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					c.Inc()
				}
			}()
		}
		wg.Wait()
		if got := c.Load(); got != 10000 {
			t.Fatalf("%s: Load = %d, ожидалось 10000", name, got)
		}
	}
}

func BenchmarkMutexCounter(b *testing.B) {
	c := &MutexCounter{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkAtomicCounter(b *testing.B) {
	c := &AtomicCounter{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}
