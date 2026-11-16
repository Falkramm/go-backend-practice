package c10

import (
	"errors"
	"sync"
	"time"

	"example.com/go-practice/internal/clock"
)

type entry struct {
	tokens     int
	lastRefill time.Time
}

// Bucket manages independent token balances for string keys.
type Bucket struct {
	mu       sync.Mutex
	mp       map[string]*entry
	c        clock.Clock
	capacity int
	rate     int
}

var (
	ErrCapacityIllegal = errors.New("illegal capacity")
	ErrRateIllegal     = errors.New("illegal rate")
)

func NewBucket(capacity int, rate int, c clock.Clock) (*Bucket, error) {
	if capacity <= 0 {
		return nil, ErrCapacityIllegal
	}

	if rate <= 0 {
		return nil, ErrRateIllegal
	}

	return &Bucket{
		mp:       make(map[string]*entry),
		c:        c,
		capacity: capacity,
		rate:     rate,
	}, nil
}

// Access spends one token on success or reports the retry delay.
func (b *Bucket) Access(key string) (allowed bool, retryAfter time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.c.Now()
	state, ok := b.mp[key]
	if !ok {
		state = &entry{tokens: b.capacity, lastRefill: now}
		b.mp[key] = state
	}

	elapsed := now.Sub(state.lastRefill)
	seconds := uint64(elapsed / time.Second)
	remainder := elapsed % time.Second
	if elapsed == time.Duration(1<<63-1) {
		// time.Sub saturates for gaps beyond the duration range.
		seconds = uint64(now.Unix()) - uint64(state.lastRefill.Unix())
		remainder = time.Duration(now.Nanosecond() - state.lastRefill.Nanosecond())
		if remainder < 0 {
			seconds--
			remainder += time.Second
		}
	}
	if seconds > 0 {
		missing := b.capacity - state.tokens
		// Clamp before multiplying to keep the refill within capacity.
		if seconds > uint64(missing/b.rate) {
			state.tokens = b.capacity
		} else {
			state.tokens += int(seconds) * b.rate
		}
		state.lastRefill = now.Add(-remainder)
	}

	if state.tokens == 0 {
		return false, time.Second
	}
	state.tokens--
	return true, 0
}
