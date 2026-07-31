package guitest

import (
	"testing"

	"github.com/kpfaulkner/guie/ui"
)

func TestCloseRequestWithoutHandlerProceeds(t *testing.T) {
	h := New(200, 120)
	h.SetContent(ui.NewContainer())
	if h.CloseHandled() {
		t.Error("an app with no close handler must leave closing to the platform")
	}
	if !h.RequestClose() {
		t.Fatal("with no handler installed the close must proceed")
	}
}

// A handler that returns false vetoes the close, and the loop must carry on: this
// is the unsaved-changes case, where the app keeps running to prompt.
func TestCloseRequestVetoKeepsLoopRunning(t *testing.T) {
	h := New(200, 120)
	label := ui.NewLabel("running")
	h.SetContent(label)

	allow := false
	calls := 0
	h.App.OnCloseRequest(func() bool {
		calls++
		return allow
	})
	if !h.CloseHandled() {
		t.Error("installing a close handler must ask the backend to defer closing")
	}

	if h.RequestClose() {
		t.Fatal("a handler returning false must veto the close")
	}
	if calls != 1 {
		t.Fatalf("handler calls = %d, want 1", calls)
	}
	if h.Err() != nil {
		t.Fatalf("a vetoed close must not error the loop: %v", h.Err())
	}
	// The app is still alive and drawing.
	label.SetText("still here")
	if rec := h.Step(); rec == nil {
		t.Fatal("Step after a vetoed close produced no frame")
	}
	if h.Err() != nil {
		t.Fatalf("stepping after a vetoed close errored: %v", h.Err())
	}

	allow = true
	if !h.RequestClose() {
		t.Fatal("a handler returning true must allow the close")
	}
	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2", calls)
	}
}

// The handler runs on the UI goroutine at the start of the frame, so work queued
// with Do before the request is visible to it -- otherwise a load or save that
// finished in the background would be missed by an unsaved-changes check.
func TestCloseRequestSeesQueuedWork(t *testing.T) {
	h := New(200, 120)
	h.SetContent(ui.NewContainer())

	dirty := false
	h.App.OnCloseRequest(func() bool { return !dirty })
	h.App.Do(func() { dirty = true })

	if h.RequestClose() {
		t.Fatal("close must be vetoed: the queued work made the app dirty")
	}
}

// Clearing the handler restores the default behaviour.
func TestCloseRequestNilClearsHandler(t *testing.T) {
	h := New(200, 120)
	h.SetContent(ui.NewContainer())
	h.App.OnCloseRequest(func() bool { return false })
	h.App.OnCloseRequest(nil)
	if h.CloseHandled() {
		t.Error("clearing the handler must hand closing back to the platform")
	}
	if !h.RequestClose() {
		t.Fatal("clearing the handler must let closes proceed")
	}
}
