package guitest_test

import (
	"fmt"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func TestTableHorizontalScrollRenders(t *testing.T) {
	h := guitest.New(260, 160)
	pal := h.App.Theme().Palette
	tb := ui.NewTable([]ui.Column{{Title: "Alpha"}, {Title: "Beta"}, {Title: "Gamma"}, {Title: "Delta"}})
	tb.SetHorizontalScroll(true) // 4 x 80 = 320 > 260
	for i := 0; i < 3; i++ {
		tb.AddRow(fmt.Sprintf("a%d", i), "b", "c", fmt.Sprintf("d%d", i))
	}
	h.SetContent(tb)
	h.MoveMouse(-1, -1)

	rec := h.Step()
	if len(rec.FillsOfColour(pal.Accent)) == 0 {
		t.Error("overflowing columns should draw a horizontal thumb")
	}
	if !rec.HasText("Alpha") || !rec.HasText("Delta") {
		t.Error("every header title is drawn (clipped, not omitted)")
	}

	// Scrolling across moves the first column's title left of where it was.
	x0 := textX(rec, "Alpha")
	h.MoveMouse(100, 100)
	h.ScrollBy(-3, 0)
	rec = h.Step()
	if x1 := textX(rec, "Alpha"); !(x1 < x0) {
		t.Errorf("after a sideways wheel the first title is at x=%v, want left of %v", x1, x0)
	}

	// The keyboard scrolls too, once the table has focus.
	tb.RequestFocus()
	h.Step()
	x1 := textX(h.Frame(), "Alpha")
	h.TypeKey(render.KeyLeft)
	if x2 := textX(h.Frame(), "Alpha"); !(x2 > x1) {
		t.Errorf("Left should scroll back: title x %v -> %v", x1, x2)
	}
}

// textX returns the x of the first drawn string equal to s.
func textX(rec *guitest.Recording, s string) float64 {
	for _, op := range rec.OpsOfKind(guitest.OpDrawText) {
		if op.Text == s {
			return op.A.X
		}
	}
	return -1e9
}
