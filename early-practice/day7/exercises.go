// День 7 — практика: sync и atomic.
//
//	cd early-practice/day7
//	go test -race -v
//	go test -bench=. -benchmem -run=^$     ← бенчмарк atomic vs mutex
package day7

import (
	"sync"
	"sync/atomic"
	"time"
)

// ============ 1. Почини Cache из мок-ревью ============
// Все четыре проблемы: value receiver, hits под RLock,
// nil-мапа без конструктора, общий *Item наружу.
// Тесты гоняют конкурентные Get/Set с -race и сверяют hits.

type Item struct {
	Value string
	Score int
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]*Item
	hits  atomic.Int32
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]*Item),
	}
}

// Get возвращает КОПИЮ элемента (чтобы вызывающие не могли
// мутировать общие данные) и признак наличия. Считает hits.
func (c *Cache) Get(key string) (Item, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item, ok := c.items[key]; ok {
		c.hits.Add(1)
		return *item, true
	}

	return Item{}, false
}

func (c *Cache) Set(key string, it Item) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &it
}

func (c *Cache) Hits() int32 {
	return c.hits.Load()
}

// ============ 2. Ленивая инициализация ============
// heavyInit дорогой и должен выполниться РОВНО ОДИН РАЗ, сколько бы
// горутин ни звало GetConfig конкурентно. Тест считает вызовы.

type Config struct{ DSN string }

// initCalls считают тесты — не трогай, просто инкрементируй в heavyInit.
var initCalls int64

func heavyInit() *Config {
	// имитация дорогой работы + счётчик вызовов:
	atomic.AddInt64(&initCalls, 1)
	time.Sleep(100 * time.Millisecond)
	return &Config{
		DSN: "HEAVY CONFIG",
	}
}

// GetConfig — конкурентно-безопасная ленивая инициализация через sync.Once
// (или OnceValue — на вкус).
var getConfig = sync.OnceValue(heavyInit)

func GetConfig() *Config {
	return getConfig()
}

// ============ 3. atomic vs mutex — почувствуй цену ============
// Два счётчика с одинаковым API. Реализуй оба, бенчмарк уже написан —
// сравни ns/op и сделай вывод (спрошу!).

type MutexCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *MutexCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}
func (c *MutexCounter) Load() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

type AtomicCounter struct {
	n atomic.Int64
}

func (c *AtomicCounter) Inc() {
	c.n.Add(1)
}
func (c *AtomicCounter) Load() int64 {
	return c.n.Load()
}
