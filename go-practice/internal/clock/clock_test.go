package clock_test

import (
	"testing"
	"time"

	"example.com/go-practice/internal/clock"
)

func TestRealNow(t *testing.T) {
	before := time.Now()
	got := (clock.Real{}).Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %v, outside [%v, %v]", got, before, after)
	}
}

func TestRealImmediateTimer(t *testing.T) {
	timer := (clock.Real{}).NewTimer(0)
	defer timer.Stop()
	select {
	case <-timer.C():
	case <-time.After(time.Second):
		t.Fatal("immediate real timer did not fire")
	}
}
