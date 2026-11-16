package c09

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/go-practice/internal/clock"
)

type Limiter struct {
	wait     chan struct{}
	next     time.Time
	interval time.Duration
	c        clock.Clock
}

var ErrIntervalIllegal = errors.New("illegal interval")

func NewLimiter(interval time.Duration, c clock.Clock) (*Limiter, error) {
	if interval <= 0 {
		return nil, ErrIntervalIllegal
	}

	return &Limiter{
		wait:     make(chan struct{}, 1),
		next:     c.Now(),
		interval: interval,
		c:        c,
	}, nil
}

func (l *Limiter) Wait(ctx context.Context) error {
	if ctx.Err() != nil {
		return fmt.Errorf("wait: %w", ctx.Err())
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait: %w", ctx.Err())
	case l.wait <- struct{}{}:
	}
	defer func() {
		<-l.wait
	}()

	delay := l.next.Sub(l.c.Now())
	if delay > 0 {
		timer := l.c.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C():
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	l.next = l.c.Now().Add(l.interval)
	return nil
}
