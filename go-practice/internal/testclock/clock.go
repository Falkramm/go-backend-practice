package testclock

import (
	"context"
	"sync"
	"time"

	"example.com/go-practice/internal/clock"
)

// Clock is a manually advanced clock for the exercise tests.
type Clock struct {
	mu      sync.Mutex
	now     time.Time
	timers  map[*timer]struct{}
	changed chan struct{}
}

func New(start time.Time) *Clock {
	return &Clock{
		now:     start,
		timers:  make(map[*timer]struct{}),
		changed: make(chan struct{}),
	}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) NewTimer(delay time.Duration) clock.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &timer{
		owner: c,
		due:   c.now.Add(delay),
		ch:    make(chan time.Time, 1),
	}
	if delay <= 0 {
		t.ch <- c.now
	} else {
		c.timers[t] = struct{}{}
	}
	c.notifyLocked()
	return t
}

// Advance moves time forward and synchronously delivers all due timer events.
func (c *Clock) Advance(delta time.Duration) {
	if delta < 0 {
		panic("testclock: cannot move time backward")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(delta)
	for t := range c.timers {
		if !t.due.After(c.now) {
			delete(c.timers, t)
			// Each timer has one buffered event and is removed before delivery.
			t.ch <- t.due
		}
	}
	c.notifyLocked()
}

func (c *Clock) ActiveTimers() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.timers)
}

// WaitForTimers waits until at least minimum timers are active.
func (c *Clock) WaitForTimers(ctx context.Context, minimum int) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		c.mu.Lock()
		if len(c.timers) >= minimum {
			c.mu.Unlock()
			return nil
		}
		changed := c.changed
		c.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Clock) notifyLocked() {
	close(c.changed)
	c.changed = make(chan struct{})
}

type timer struct {
	owner *Clock
	due   time.Time
	ch    chan time.Time
}

func (t *timer) C() <-chan time.Time {
	return t.ch
}

func (t *timer) Stop() {
	c := t.owner
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, active := c.timers[t]; active {
		delete(c.timers, t)
		c.notifyLocked()
	}
}
