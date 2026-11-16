// День 2 — практика: мапы, интерфейсы, конкурентный доступ.
//
//	cd early-practice/day2
//	go test -race -v        ← -race сегодня обязателен!
package day2

import "errors"
import "strings"
import "sync"
import "maps"

// WordCount возвращает мапу слово → сколько раз встретилось.
// Слова разделены пробелами, регистр не трогаем. "" -> пустая (не nil!) мапа.
func WordCount(s string) map[string]int {
	m := make(map[string]int)
	for _, w := range strings.Fields(s) {
		m[w]++
	}
	return m
}

// TopWord возвращает слово с максимальным счётчиком из counts и его счётчик.
// При равенстве счётчиков — лексикографически меньшее слово (подумай,
// почему без этого правила функция была бы недетерминированной).
// Пустая/nil мапа -> ("", 0).
func TopWord(counts map[string]int) (string, int) {
	var word string
	var count int
	for k, v := range counts {
		if v > count || v == count && k < word {
			word = k
			count = v
		}
	}
	return word, count
}

// ErrNotFound — сентинел для Lookup.
var ErrNotFound = errors.New("not found")

// Lookup возвращает значение по ключу. Ключа нет -> ошибка ErrNotFound.
// ВАЖНО: ключ со значением 0 — это НЕ отсутствие ключа (comma-ok!).
func Lookup(m map[string]int, key string) (int, error) {
	v, ok := m[key]
	if !ok {
		return v, ErrNotFound
	} else {
		return v, nil
	}
}

// --- Typed nil: почини функцию ---

type ParseErr struct{ Line int }

func (e *ParseErr) Error() string { return "parse error" }

// ParseAll обходит строки и возвращает первую ошибку парсинга (строка "bad").
// Сейчас функция сломана классическим typed nil — почини её так,
// чтобы при отсутствии "bad" она возвращала честный nil.
func ParseAll(lines []string) error {
	for i, l := range lines {
		if l == "bad" {
			return &ParseErr{Line: i}
		}
	}
	return nil
}

// --- Конкурентность: потокобезопасный счётчик ---

// Counter — потокобезопасный счётчик строковых ключей.
// Заполни структуру и методы так, чтобы go test -race был доволен.
// Inc можно звать из многих горутин; Get и Snapshot — тоже.
type Counter struct {
	m  map[string]int
	mu sync.RWMutex
}

func NewCounter() *Counter {
	return &Counter{
		m:  make(map[string]int),
		mu: sync.RWMutex{},
	}
}

// Inc увеличивает счётчик ключа на 1.
func (c *Counter) Inc(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[key]++
}

// Get возвращает текущее значение ключа (0, если ключа нет).
func (c *Counter) Get(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.m[key]
}

// Snapshot возвращает КОПИЮ всех счётчиков.
// Подумай: почему нельзя просто вернуть внутреннюю мапу?
func (c *Counter) Snapshot() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return maps.Clone(c.m)
}
