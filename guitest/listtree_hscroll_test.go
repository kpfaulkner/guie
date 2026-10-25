package guitest_test

import (
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

func TestListScrollsAcrossWhenAsked(t *testing.T) {
	long := "LONG" + strings.Repeat("-", 120)

	h := guitest.New(200, 120)
	pal := h.App.Theme().Palette
	l := ui.NewList([]string{"short", long})
	l.SetHorizontalScroll(true)
	h.SetContent(l)
	h.MoveMouse(-1, -1)
	rec := h.Step()
	if len(rec.FillsOfColour(pal.Accent)) == 0 {
		t.Error("a row wider than the list should draw a horizontal thumb")
	}
	x0 := lineX(rec, "LONG")

	h.MoveMouse(60, 30)
	h.ScrollBy(-6, 0)
	if x1 := lineX(h.Step(), "LONG"); !(x1 < x0-100) {
		t.Errorf("after a sideways wheel the long row is at x=%v, want well left of %v", x1, x0)
	}

	// Off (the default), the same list shows no bar and clips the row.
	h2 := guitest.New(200, 120)
	l2 := ui.NewList([]string{"short", long})
	h2.SetContent(l2)
	h2.MoveMouse(-1, -1)
	if n := len(h2.Step().FillsOfColour(pal.Accent)); n != 0 {
		t.Errorf("default list drew %d accent fills; it has no bar to show", n)
	}
}

func TestTreeScrollsAcrossWhenAsked(t *testing.T) {
	h := guitest.New(200, 120)
	pal := h.App.Theme().Palette
	deep := ui.NewTreeNode("DEEP" + strings.Repeat("-", 120))
	root := ui.NewTreeNode("root", ui.NewTreeNode("mid", deep))
	tr := ui.NewTree(root)
	tr.SetHorizontalScroll(true)
	tr.Expand(root)
	tr.Expand(root.Children()[0])
	h.SetContent(tr)
	h.MoveMouse(-1, -1)
	rec := h.Step()
	if len(rec.FillsOfColour(pal.Accent)) == 0 {
		t.Error("a row wider than the tree should draw a horizontal thumb")
	}
	x0 := lineX(rec, "DEEP")

	h.MoveMouse(60, 30)
	h.ScrollBy(-6, 0)
	if x1 := lineX(h.Step(), "DEEP"); !(x1 < x0-100) {
		t.Errorf("after a sideways wheel the deep row is at x=%v, want well left of %v", x1, x0)
	}
}
