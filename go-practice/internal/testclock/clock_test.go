package testclock_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"example.com/go-practice/internal/testclock"
)

func TestManualStartAndAdvance(t *testing.T) {
	start := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	c := testclock.New(start)
	if got := c.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}
	c.Advance(500 * time.Millisecond)
	if got := c.Now(); !got.Equal(start.Add(500 * time.Millisecond)) {
		t.Fatalf("Now() after Advance = %v", got)
	}
}

func TestManualTimerDeadline(t *testing.T) {
	start := time.Unix(0, 0)
	c := testclock.New(start)
	timer := c.NewTimer(time.Second)
	if got := c.ActiveTimers(); got != 1 {
		t.Fatalf("ActiveTimers() = %d, want 1", got)
	}
	c.Advance(time.Second - time.Nanosecond)
	select {
	case <-timer.C():
		t.Fatal("timer fired before its deadline")
	default:
	}
	c.Advance(time.Nanosecond)
	select {
	case firedAt := <-timer.C():
		if !firedAt.Equal(start.Add(time.Second)) {
			t.Fatalf("timer timestamp = %v", firedAt)
		}
	default:
		t.Fatal("timer did not fire at its deadline")
	}
	if got := c.ActiveTimers(); got != 0 {
		t.Fatalf("ActiveTimers() after firing = %d, want 0", got)
	}
	c.Advance(time.Second)
	select {
	case <-timer.C():
		t.Fatal("one-shot timer fired twice")
	default:
	}
}

func TestManualTimerStop(t *testing.T) {
	c := testclock.New(time.Unix(0, 0))
	timer := c.NewTimer(time.Second)
	timer.Stop()
	timer.Stop()
	if got := c.ActiveTimers(); got != 0 {
		t.Fatalf("ActiveTimers() = %d, want 0", got)
	}
	c.Advance(2 * time.Second)
	select {
	case <-timer.C():
		t.Fatal("stopped timer fired")
	default:
	}
}

func TestManualImmediateTimer(t *testing.T) {
	for _, delay := range []time.Duration{0, -time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			start := time.Unix(0, 0)
			c := testclock.New(start)
			timer := c.NewTimer(delay)
			select {
			case got := <-timer.C():
				if !got.Equal(start) {
					t.Fatalf("immediate timer timestamp = %v", got)
				}
			default:
				t.Fatal("nonpositive timer did not fire immediately")
			}
			if got := c.ActiveTimers(); got != 0 {
				t.Fatalf("ActiveTimers() = %d, want 0", got)
			}
		})
	}
}

func TestWaitForTimers(t *testing.T) {
	c := testclock.New(time.Unix(0, 0))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- c.WaitForTimers(ctx, 1) }()
	timer := c.NewTimer(time.Hour)
	defer timer.Stop()
	if err := <-finished; err != nil {
		t.Fatalf("WaitForTimers returned %v", err)
	}
}

func TestWaitForTimersCanceled(t *testing.T) {
	c := testclock.New(time.Unix(0, 0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.WaitForTimers(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("WaitForTimers returned %v, want context.Canceled", err)
	}
}

func TestManualConcurrentAccess(t *testing.T) {
	c := testclock.New(time.Unix(0, 0))
	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for attempt := 0; attempt < 32; attempt++ {
				_ = c.Now()
				timer := c.NewTimer(time.Second)
				timer.Stop()
			}
		}()
	}
	for step := 0; step < 32; step++ {
		c.Advance(time.Millisecond)
	}
	workers.Wait()
	if got := c.ActiveTimers(); got != 0 {
		t.Fatalf("ActiveTimers() = %d, want 0", got)
	}
}

func TestManualRejectsBackwardTime(t *testing.T) {
	c := testclock.New(time.Unix(0, 0))
	defer func() {
		if recover() == nil {
			t.Fatal("Advance with a negative duration should panic")
		}
	}()
	c.Advance(-time.Nanosecond)
}
