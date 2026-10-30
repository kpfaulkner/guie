package ui

import (
	"testing"
	"time"
)

// settle runs queued results until no Async job is left, failing if one hangs.
func settle(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for a.AsyncPending() > 0 {
		if time.Now().After(deadline) {
			t.Fatal("Async work did not finish")
		}
		a.runPending()
		time.Sleep(50 * time.Microsecond)
	}
}

func TestAppAsyncRunsWorkOffThreadAndDoneOnUI(t *testing.T) {
	a := NewApp()
	release := make(chan struct{})
	var got, doneRan int
	a.Async(func() { <-release; got = 42 }, func() { doneRan++ })

	if a.AsyncPending() != 1 {
		t.Fatalf("pending = %d, want 1", a.AsyncPending())
	}
	a.runPending()
	if doneRan != 0 {
		t.Fatal("done ran before the work finished")
	}
	close(release)
	settle(t, a)
	if doneRan != 1 || got != 42 {
		t.Errorf("done ran %d times, got = %d; want 1 and 42", doneRan, got)
	}
	if a.AsyncPending() != 0 {
		t.Errorf("pending = %d after settling, want 0", a.AsyncPending())
	}
}

func TestAppAsyncDoneRunsOnlyWhenFramesRun(t *testing.T) {
	a := NewApp()
	ran := false
	a.Async(func() {}, func() { ran = true })
	deadline := time.Now().Add(5 * time.Second)
	for {
		a.mu.Lock()
		queued := len(a.pending)
		a.mu.Unlock()
		if queued > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("result was never queued")
		}
		time.Sleep(50 * time.Microsecond)
	}
	if ran {
		t.Error("done must wait for the UI goroutine to drain the queue")
	}
	if a.AsyncPending() != 1 {
		t.Errorf("a queued result still counts as pending, got %d", a.AsyncPending())
	}
	a.runPending()
	if !ran || a.AsyncPending() != 0 {
		t.Errorf("after the drain: ran=%v pending=%d", ran, a.AsyncPending())
	}
}

func TestAppAsyncChainedAndNilArguments(t *testing.T) {
	a := NewApp()
	a.Async(nil, func() { t.Error("done of a nil work must not run") })
	if a.AsyncPending() != 0 {
		t.Fatalf("a nil work started a job: pending = %d", a.AsyncPending())
	}

	var order []string
	a.Async(func() {}, func() {
		order = append(order, "first")
		// done may start more work; the count must not touch zero in between.
		a.Async(func() {}, func() { order = append(order, "second") })
	})
	a.Async(func() {}, nil) // a nil done just runs the work
	settle(t, a)
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("order = %v, want [first second]", order)
	}
}
