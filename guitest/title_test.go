package guitest

import "testing"

func TestSetTitleReachesDriver(t *testing.T) {
	h := New(200, 100)
	h.Step()
	if got := h.Title(); got != "" {
		t.Fatalf("title before any change = %q, want empty", got)
	}
	h.App.SetTitle("hello")
	if got := h.Title(); got != "hello" {
		t.Errorf("title = %q, want hello", got)
	}
	h.App.SetTitle("")
	if got := h.Title(); got != "" {
		t.Errorf("title after clearing = %q", got)
	}
}
