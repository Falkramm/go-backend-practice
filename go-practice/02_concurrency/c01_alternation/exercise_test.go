package c01

import (
	"slices"
	"testing"
	"time"
)

func TestAlternate(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want []int
	}{
		{name: "negative", n: -3, want: nil},
		{name: "zero", n: 0, want: nil},
		{name: "one", n: 1, want: []int{1}},
		{name: "two", n: 2, want: []int{1, 2}},
		{name: "odd", n: 5, want: []int{1, 2, 3, 4, 5}},
		{name: "even", n: 6, want: []int{1, 2, 3, 4, 5, 6}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := alternateWithTimeout(t, test.n)
			if !slices.Equal(got, test.want) {
				t.Fatalf("Alternate(%d) = %v, want %v", test.n, got, test.want)
			}
		})
	}
}

func TestAlternateRepeated(t *testing.T) {
	for attempt := 0; attempt < 200; attempt++ {
		n := 1 + attempt%32
		got := alternateWithTimeout(t, n)
		if len(got) != n {
			t.Fatalf("run %d: Alternate(%d) returned %d values, want %d", attempt, n, len(got), n)
		}

		for index, value := range got {
			if value != index+1 {
				t.Fatalf("run %d: Alternate(%d)[%d] = %d, want %d", attempt, n, index, value, index+1)
			}
		}
	}
}

func alternateWithTimeout(t *testing.T, n int) []int {
	t.Helper()

	completed := make(chan []int, 1)
	go func() {
		completed <- Alternate(n)
	}()

	const timeout = 2 * time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-completed:
		return result
	case <-timer.C:
		t.Fatalf("Alternate(%d) did not return within %s", n, timeout)
		return nil
	}
}
