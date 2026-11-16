package c08

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewCounter(t *testing.T) {
	counter := NewCounter()
	if counter == nil {
		t.Fatal("NewCounter returned nil")
	}
	if got := counter.Get("missing"); got != 0 {
		t.Fatalf("Get(missing) = %d, want 0", got)
	}
	if got := counter.Snapshot(); len(got) != 0 {
		t.Fatalf("initial Snapshot = %v, want empty", got)
	}
}

func TestCounterAddAndGet(t *testing.T) {
	counter := NewCounter()
	counter.Add("views", 3)
	counter.Add("views", -1)
	counter.Add("sales", 7)
	counter.Add("sales", 0)
	if got := counter.Get("views"); got != 2 {
		t.Errorf("views = %d, want 2", got)
	}
	if got := counter.Get("sales"); got != 7 {
		t.Errorf("sales = %d, want 7", got)
	}
	if got := counter.Get("missing"); got != 0 {
		t.Errorf("missing = %d, want 0", got)
	}
}

func TestCounterConcurrentUpdatesAndReads(t *testing.T) {
	const writers = 16
	const increments = 250
	counter := NewCounter()
	var group sync.WaitGroup
	start := make(chan struct{})
	for writer := 0; writer < writers; writer++ {
		group.Add(1)
		go func(id int) {
			defer group.Done()
			<-start
			key := fmt.Sprintf("writer-%d", id)
			for i := 0; i < increments; i++ {
				counter.Add("shared", 1)
				counter.Add("negative", -2)
				counter.Add(key, 1)
			}
		}(writer)
	}
	for reader := 0; reader < 4; reader++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			for i := 0; i < increments; i++ {
				counter.Get("shared")
				counter.Get("missing")
				counter.Snapshot()
			}
		}()
	}
	close(start)
	finished := make(chan struct{})
	go func() {
		group.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent operations did not finish")
	}
	if got := counter.Get("shared"); got != writers*increments {
		t.Errorf("shared = %d, want %d", got, writers*increments)
	}
	if got := counter.Get("negative"); got != -2*writers*increments {
		t.Errorf("negative = %d, want %d", got, -2*writers*increments)
	}
	for writer := 0; writer < writers; writer++ {
		key := fmt.Sprintf("writer-%d", writer)
		if got := counter.Get(key); got != increments {
			t.Errorf("%s = %d, want %d", key, got, increments)
		}
	}
}

func TestSnapshotIsIndependent(t *testing.T) {
	counter := NewCounter()
	counter.Add("views", 7)
	snapshot := counter.Snapshot()
	if !reflect.DeepEqual(snapshot, map[string]int{"views": 7}) {
		t.Fatalf("Snapshot = %v, want views=7", snapshot)
	}
	snapshot["views"] = 99
	snapshot["outside"] = 1
	if got := counter.Get("views"); got != 7 {
		t.Errorf("snapshot mutation changed views to %d", got)
	}
	if got := counter.Get("outside"); got != 0 {
		t.Errorf("snapshot mutation added internal key outside=%d", got)
	}
	counter.Add("views", 1)
	if snapshot["views"] != 99 {
		t.Errorf("later Add changed the previous snapshot: %v", snapshot)
	}
	if got := counter.Snapshot(); !reflect.DeepEqual(got, map[string]int{"views": 8}) {
		t.Errorf("new Snapshot = %v, want views=8", got)
	}
}

func TestSnapshotAfterEachCompletedUpdate(t *testing.T) {
	counter := NewCounter()
	steps := []struct {
		key   string
		delta int
		want  map[string]int
	}{
		{key: "a", delta: 1, want: map[string]int{"a": 1}},
		{key: "b", delta: 2, want: map[string]int{"a": 1, "b": 2}},
		{key: "a", delta: 2, want: map[string]int{"a": 3, "b": 2}},
		{key: "b", delta: -1, want: map[string]int{"a": 3, "b": 1}},
	}
	for _, step := range steps {
		counter.Add(step.key, step.delta)
		if got := counter.Snapshot(); !reflect.DeepEqual(got, step.want) {
			t.Fatalf("after Add(%q, %d): Snapshot=%v, want %v", step.key, step.delta, got, step.want)
		}
	}
}
