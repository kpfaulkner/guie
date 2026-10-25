package guitest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func areaWith(h *guitest.Harness, text string, opts ...ui.TextAreaOption) *ui.TextArea {
	ta := ui.NewTextArea(opts...)
	ta.SetText(text)
	ta.SelectRange(0, 0) // SetText leaves the caret at the end; start at the top
	h.SetContent(ta)
	h.MoveMouse(-1, -1)
	return ta
}

func TestTextAreaDrawsHorizontalBarForLongLines(t *testing.T) {
	h := guitest.New(260, 140)
	pal := h.App.Theme().Palette
	areaWith(h, strings.Repeat("0123456789", 40))
	if len(h.Step().FillsOfColour(pal.Accent)) == 0 {
		t.Error("a line wider than the view should draw a horizontal thumb")
	}

	// Wrapped, the same kind of line breaks at the edge into a few rows that fit, so
	// neither bar is needed.
	h2 := guitest.New(260, 140)
	areaWith(h2, strings.Repeat("0123456789", 12), ui.TextAreaWrap())
	if n := len(h2.Step().FillsOfColour(pal.Accent)); n != 0 {
		t.Errorf("wrapped text drew %d accent fills; it has no scrollbar to show", n)
	}
}

func TestTextAreaScrollsAcrossOnWheel(t *testing.T) {
	h := guitest.New(260, 140)
	areaWith(h, "start "+strings.Repeat("0123456789", 40))
	x0 := lineX(h.Step(), "start")
	if x0 < 0 {
		t.Fatalf("the line should be drawn inside the box before scrolling, x=%v", x0)
	}
	h.MoveMouse(100, 60)
	h.ScrollBy(-8, 0) // sideways wheel
	if x1 := lineX(h.Step(), "start"); !(x1 < x0-100) {
		t.Errorf("after scrolling across the line is at x=%v, want it moved well left of %v", x1, x0)
	}
}

// The thumb of a scrollbar must stay where the user drags it. The view used to
// snap back to the caret every frame, which only showed up once a bar could move
// the view: the caret stays at the top while the thumb is dragged away.
func TestTextAreaThumbDragSticksAcrossFrames(t *testing.T) {
	h := guitest.New(260, 140)
	var b strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "row%03d\n", i)
	}
	areaWith(h, b.String())
	if !h.Step().HasText("row000") {
		t.Fatal("the first row should show at the start")
	}

	// Drag the vertical thumb (right edge, near the top) a good way down.
	h.Drag(254, 10, 254, 80)
	h.Step()
	rec := h.Step() // a further frame must not undo the scroll
	if rec.HasText("row000") {
		t.Error("the view snapped back to the caret after the thumb was dragged")
	}
}

// lineX returns the x of the first drawn string containing s (-1e9 if none). A
// scrolled line is still drawn, just clipped, so its position is what moves.
func lineX(rec *guitest.Recording, s string) float64 {
	for _, op := range rec.OpsOfKind(guitest.OpDrawText) {
		if strings.Contains(op.Text, s) {
			return op.A.X
		}
	}
	return -1e9
}
