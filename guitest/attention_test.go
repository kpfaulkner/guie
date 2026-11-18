package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestWindowFocusAndAttention(t *testing.T) {
	h := guitest.New(200, 100)
	h.SetContent(ui.NewLabel("x"))
	h.Step()
	if !h.App.WindowFocused() {
		t.Fatal("a new window is not focused")
	}
	h.SetWindowFocused(false)
	if h.App.WindowFocused() {
		t.Fatal("still focused")
	}
	if !h.App.RequestAttention() || !h.App.RequestAttention() || h.AttentionRequests() != 2 {
		t.Fatalf("requests %d", h.AttentionRequests())
	}
}

func TestAttentionBeforeRun(t *testing.T) {
	a := ui.NewApp()
	if !a.WindowFocused() || a.RequestAttention() {
		t.Fatal("before Run the window counts as focused and attention is refused")
	}
}
