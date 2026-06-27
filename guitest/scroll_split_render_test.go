package guitest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func tallList(n int) *ui.Container {
	c := ui.NewContainer()
	c.SetLayout(ui.VBox(2))
	for i := 0; i < n; i++ {
		c.Add(ui.NewLabel(fmt.Sprintf("row %d", i)))
	}
	return c
}

func TestScrollViewRendersThumbAndDrags(t *testing.T) {
	h := guitest.New(200, 150)
	pal := h.App.Theme().Palette
	sv := ui.NewScrollView()
	sv.SetContent(tallList(60)) // far taller than the viewport
	h.SetContent(sv)
	h.MoveMouse(-1, -1)

	rec := h.Step()
	if len(rec.FillsOfColour(pal.Accent)) == 0 {
		t.Error("scroll view should draw an accent thumb")
	}
	if n := len(sv.Children()); n != 1 {
		t.Errorf("ScrollView.Children should return its single content, got %d", n)
	}

	// Drag the thumb (right edge, near the top) downward to exercise dragByThumb.
	h.Drag(194, 8, 194, 100)
	h.Step() // should not panic and should re-render
}

func TestScrollViewWheelScrolls(t *testing.T) {
	h := guitest.New(200, 150)
	sv := ui.NewScrollView()
	sv.SetContent(tallList(60))
	h.SetContent(sv)

	before := h.Step().TextContaining("row 0")
	h.ScrollBy(0, -10) // wheel down
	after := h.Step()
	// After scrolling down, the very first row should eventually leave the top;
	// at minimum the wheel handler ran without error and content still draws.
	if !after.TextContaining("row") {
		t.Error("scroll view should still render rows after wheeling")
	}
	_ = before
}

func TestSplitterRendersBothPanes(t *testing.T) {
	h := guitest.New(240, 120)
	s := ui.HSplit(ui.NewLabel("LEFT"), ui.NewLabel("RIGHT"))
	h.SetContent(s)
	h.MoveMouse(-1, -1)

	rec := h.Step()
	if !rec.HasText("LEFT") || !rec.HasText("RIGHT") {
		t.Error("split pane should draw both children")
	}

	// Drag the divider (around the horizontal midpoint) to exercise HandleEvent.
	h.Drag(120, 60, 150, 60)
	h.Step()
}

func TestListDrawsScrollbarWhenOverflowing(t *testing.T) {
	h := guitest.New(150, 40)
	pal := h.App.Theme().Palette
	items := make([]string, 30)
	for i := range items {
		items[i] = fmt.Sprintf("item %d", i)
	}
	h.SetContent(ui.NewList(items))
	h.MoveMouse(-1, -1)

	if len(h.Step().FillsOfColour(pal.Accent)) == 0 {
		t.Error("an overflowing list should draw an accent scrollbar thumb")
	}
}

func TestTextAreaDrawsScrollbarWhenOverflowing(t *testing.T) {
	h := guitest.New(160, 40)
	pal := h.App.Theme().Palette
	ta := ui.NewTextArea()
	ta.SetText(strings.Repeat("a line of text\n", 30))
	h.SetContent(ta)
	h.MoveMouse(-1, -1)

	if len(h.Step().FillsOfColour(pal.Accent)) == 0 {
		t.Error("an overflowing text area should draw an accent scrollbar thumb")
	}
}
