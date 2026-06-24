package ui_test

import (
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestDebugBoundsOutlinesEveryWidget(t *testing.T) {
	h := guitest.New(200, 100)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(4))
	root.Add(ui.NewLabel("a"))
	root.Add(ui.NewLabel("b"))
	h.SetContent(root)

	// Off by default: the bare container + labels draw no rect outlines.
	base := h.Step().Count(guitest.OpStrokeRect)
	if base != 0 {
		t.Fatalf("debug off: want 0 stroke-rects, got %d", base)
	}

	// On: one outline per widget (root + its two children = 3).
	h.App.SetDebugBounds(true)
	if got := h.Step().Count(guitest.OpStrokeRect); got != 3 {
		t.Fatalf("debug on: want 3 stroke-rects, got %d", got)
	}

	// Each widget gets its own colour, so the three outlines are all different.
	seen := map[color.Color]bool{}
	for _, op := range h.Step().OpsOfKind(guitest.OpStrokeRect) {
		seen[op.Colour] = true
	}
	if len(seen) != 3 {
		t.Fatalf("want 3 distinct outline colours, got %d", len(seen))
	}
}
