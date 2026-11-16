// 🏗 System Design
// Теперь самое интересное. Первая задача — классика Go-интервью.

// Задача: Rate Limiter
// Спроектируй и реализуй rate limiter для HTTP API.

// Требования:

// Ограничить: не более 100 запросов в секунду на каждый API-ключ
// Работает в многопоточной среде (сотни горутин)
// Должен быть эффективным — не блокировать всё на одном мьютексе
// Ключей может быть много (десятки тысяч), нужно чистить неактивные
// Что хочу услышать:

// Какой алгоритм выберешь? Знаешь варианты — token bucket, leaky bucket, fixed window, sliding window? Плюсы/минусы, что подходит здесь?

// Структуры данных — как хранить состояние по ключам? (вспомни блок про мапы и конкурентность)

// Синхронизация — mutex / RWMutex / sync.Map / шардирование / atomic? Обоснуй.

// Утечка памяти — что будет с мапой через сутки работы? Как чистить?

// Код — набросай основную структуру и метод Allow(key string) bool

// Начни с пункта 1 — выбор алгоритма и почему. 👇

type bucket struct {
	mu sync.Mutex
	tokens float64
	last time.Time
}

func (b *bucket) allow(rate, burst float64) (bool, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	b.tokens = min(burst, b.tokens + elapsed * rate)

	if b.tokens >= 1 {
		b.tokens--
		return true, time.Duration(0)
	}
	return false, time.Duration(1 - b.tokens) / rate * time.Second
}

type shard struct {
	mu sync.RWMutex
	buckets map[string]*bucket
}

const shardsCount = 256

type Limiter struct {
	shards [shardsCount]*shard
	rate float64
	burst float64
}

func (l *Limiter) junitor(ttl time.Duration) {
	ticker := time.NewTicker(ttl / 2)
	for range ticker.C {
		cuttoff := time.Now().Add(-ttl)
		for _, s := range l.shards {
			s.mu.Lock()
			for key, bucket := range s.buckets {
				b.mu.Lock()
				stale := b.last.Before(cuttoff)
				b.mu.Unlock()
				if stale {
					delete(s.buckets, key)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (l *Limiter) getShard(key string) *shard {
	h := fnv32(key)
	return l.shards[h&(shardsCount - 1)]
}

func (l *Limiter) Allow(key string) (bool, time.Duration) {
	s := l.getShard(key)
	s.mu.RLock()
	b, ok := s.buckets[key]
	s.mu.RUnlock()

	if !ok {
		s.mu.Lock()
		b, ok = s.buckets[key]
		if !ok {
			b = &bucket{
				tokens : l.burst,
				last : time.Now()
			}
			s.buckets[key] = b
		}
		s.mu.Unlock()
	}

	return b.allow(l.rate, l.burst)
}
