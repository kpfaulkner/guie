package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

// openContextMenu right-clicks a widget that fills the window and has items
// as its context menu.
func openContextMenu(t *testing.T, items ...ui.MenuItem) *guitest.Harness {
	t.Helper()
	h := guitest.New(400, 300)
	w := ui.NewContainer()
	w.SetContextMenu(items...)
	h.SetContent(w)
	h.Step()
	h.RightClick(20, 20)
	return h
}

func textOp(t *testing.T, r *guitest.Recording, s string) guitest.Op {
	t.Helper()
	for _, op := range r.OpsOfKind(guitest.OpDrawText) {
		if op.Text == s {
			return op
		}
	}
	t.Fatalf("no %q drawn; have %v", s, r.Texts())
	return guitest.Op{}
}

func TestDisabledMenuItemIsGreyedOutAndCannotBeChosen(t *testing.T) {
	ran := ""
	h := openContextMenu(t,
		ui.NewMenuItem("Copy", func() { ran = "Copy" }),
		ui.MenuItem{Label: "Paste", Action: func() { ran = "Paste" }, Disabled: true},
	)
	pal := h.App.Theme().Palette
	r := h.Step()
	if !sameColour(textOp(t, r, "Copy").Colour, pal.Text) {
		t.Fatal("an enabled item should be drawn in the text colour")
	}
	paste := textOp(t, r, "Paste")
	if !sameColour(paste.Colour, pal.TextMuted) {
		t.Fatal("a disabled item should be drawn in the muted colour")
	}

	// Pointing at it does not highlight it.
	h.MoveMouse(paste.A.X+2, paste.A.Y+2)
	if fills := h.Step().FillsOfColour(pal.Accent); len(fills) != 0 {
		t.Fatalf("a disabled item was highlighted: %v", fills)
	}

	// Clicking it does nothing, and the menu stays open.
	h.Click(paste.A.X+2, paste.A.Y+2)
	r = h.Step()
	if ran != "" {
		t.Fatalf("ran %q", ran)
	}
	if !r.HasText("Paste") {
		t.Fatal("clicking a disabled item closed the menu")
	}

	// The other item still works.
	copyOp := textOp(t, r, "Copy")
	h.MoveMouse(copyOp.A.X+2, copyOp.A.Y+2)
	if fills := h.Step().FillsOfColour(pal.Accent); len(fills) != 1 {
		t.Fatalf("an enabled item was not highlighted: %v", fills)
	}
	h.Click(copyOp.A.X+2, copyOp.A.Y+2)
	r = h.Step()
	if ran != "Copy" || r.HasText("Paste") {
		t.Fatalf("ran %q; texts %v", ran, r.Texts())
	}
}

func TestMenuSeparatorDrawsALineAndLeavesRowsClickable(t *testing.T) {
	ran := ""
	h := openContextMenu(t,
		ui.NewMenuItem("Above", func() { ran = "Above" }),
		ui.NewMenuSeparator(),
		ui.NewMenuItem("Below", func() { ran = "Below" }),
	)
	pal := h.App.Theme().Palette
	r := h.Step()
	above, below := textOp(t, r, "Above"), textOp(t, r, "Below")

	var line *guitest.Op
	for _, op := range r.OpsOfKind(guitest.OpDrawLine) {
		if sameColour(op.Colour, pal.Border) && op.A.Y == op.B.Y && op.A.Y > above.A.Y && op.A.Y < below.A.Y {
			line = &op
		}
	}
	if line == nil {
		t.Fatalf("no separator line between the items: %v", r.OpsOfKind(guitest.OpDrawLine))
	}
	if line.B.X <= line.A.X {
		t.Fatalf("line %v to %v", line.A, line.B)
	}

	// Clicking the separator does nothing and leaves the menu open.
	h.Click(line.A.X+4, line.A.Y)
	r = h.Step()
	if ran != "" || !r.HasText("Below") {
		t.Fatalf("ran %q; texts %v", ran, r.Texts())
	}

	// The item below it is where it is drawn.
	below = textOp(t, r, "Below")
	h.Click(below.A.X+2, below.A.Y+2)
	h.Step()
	if ran != "Below" {
		t.Fatalf("ran %q", ran)
	}
}

func TestMenuBarHasDisabledItemsAndSeparators(t *testing.T) {
	h := guitest.New(300, 200)
	ran := 0
	mb := ui.NewMenuBar()
	mb.AddMenu("File",
		ui.NewMenuItem("Open", func() {}),
		ui.NewMenuSeparator(),
		ui.MenuItem{Label: "Save", Action: func() { ran++ }, Disabled: true},
	)
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(mb)
	root.Add(ui.NewContainer(), ui.Weight(1))
	h.SetContent(root)
	h.MoveMouse(-1, -1)
	h.Step()

	bar := mb.Bounds()
	h.Click(bar.X+8, bar.Y+bar.H/2)
	r := h.Step()
	save := textOp(t, r, "Save")
	if !sameColour(save.Colour, h.App.Theme().Palette.TextMuted) {
		t.Fatal("a disabled item should be drawn in the muted colour")
	}
	h.Click(save.A.X+2, save.A.Y+2)
	if r = h.Step(); ran != 0 || !r.HasText("Save") {
		t.Fatalf("ran %d; texts %v", ran, r.Texts())
	}
}
