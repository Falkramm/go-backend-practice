package clock

import "time"

// Timer exposes the operations needed by the timed exercises.
type Timer interface {
	C() <-chan time.Time
	Stop()
}

// Clock supplies the current time and one-shot timers.
type Clock interface {
	Now() time.Time
	NewTimer(time.Duration) Timer
}

// Real uses the standard library clock.
type Real struct{}

func (Real) Now() time.Time {
	return time.Now()
}

func (Real) NewTimer(delay time.Duration) Timer {
	return realTimer{timer: time.NewTimer(delay)}
}

type realTimer struct {
	timer *time.Timer
}

func (t realTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t realTimer) Stop() {
	t.timer.Stop()
}
