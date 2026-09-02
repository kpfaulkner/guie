package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

// Rebuilding part of the tree must not leave the keyboard pointed at a widget
// that is no longer in it.
//
// Focus is a pointer, and removing a widget from its parent does not go through
// the App, so every keystroke went on being delivered to the detached widget:
// the field on screen looked like it had stopped accepting input, while what the
// user typed accumulated somewhere invisible.
func TestFocusIsDroppedWhenTheFocusedWidgetIsRemoved(t *testing.T) {
	h := guitest.New(300, 200)

	panel := ui.NewContainer()
	panel.SetLayout(ui.VBox(0))
	old := ui.NewTextField()
	panel.Add(old, ui.Align(geom.AlignStretch))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(panel, ui.Weight(1))
	h.SetContent(root)
	h.Step()

	centre := old.Bounds().Center()
	h.Click(centre.X, centre.Y)
	h.TypeText("abc")
	h.Step()
	if old.Text() != "abc" {
		t.Fatalf("setup: field holds %q, want %q", old.Text(), "abc")
	}

	// Rebuild the panel, exactly as a refreshed table does.
	panel.Remove(old)
	replacement := ui.NewTextField()
	panel.Add(replacement, ui.Align(geom.AlignStretch))
	h.Step()

	h.TypeText("xyz")
	h.Step()

	if got := old.Text(); got != "abc" {
		t.Errorf("the removed field received %q; it is no longer in the tree and must get nothing", got)
	}
	if got := replacement.Text(); got != "" {
		t.Errorf("the new field holds %q; with focus cleared it should receive nothing until it is clicked", got)
	}

	// And it still accepts input once clicked.
	centre = replacement.Bounds().Center()
	h.Click(centre.X, centre.Y)
	h.TypeText("xyz")
	h.Step()
	if got := replacement.Text(); got != "xyz" {
		t.Errorf("after clicking it the new field holds %q, want %q", got, "xyz")
	}
}

// Focus that is still attached must be left alone - the check must not clear
// focus on every layout pass.
func TestFocusSurvivesALayoutThatKeepsTheWidget(t *testing.T) {
	h := guitest.New(300, 200)

	field := ui.NewTextField()
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(field, ui.Align(geom.AlignStretch))
	h.SetContent(root)
	h.Step()

	centre := field.Bounds().Center()
	h.Click(centre.X, centre.Y)
	h.TypeText("abc")
	h.Step()

	// A layout pass that does not detach the field: add a sibling, resize.
	root.Add(ui.NewLabel("sibling"))
	h.Step()
	h.Resize(500, 200)
	h.Step()

	h.TypeText("def")
	h.Step()

	if got := field.Text(); got != "abcdef" {
		t.Errorf("field holds %q, want %q: focus was dropped even though the widget is still in the tree", got, "abcdef")
	}
}
