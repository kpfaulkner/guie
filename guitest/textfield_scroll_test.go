package guitest_test

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

// A field whose text overflowed while it was narrow must show that text from
// the start once it is wide enough to hold it.
//
// The horizontal offset only ever grew to keep the caret visible and was never
// reduced when the field gained room, so a field laid out narrow and later
// widened - a resized window, or a weighted table cell that starts at its
// minimum width - drew its text off to the left of the box, clipped mid-string.
func TestTextScrollsBackWhenTheFieldIsWidened(t *testing.T) {
	h := guitest.New(80, 100)

	field := ui.NewTextField()
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(field, ui.Align(geom.AlignStretch))
	h.SetContent(root)

	h.Step()
	field.SetText(`c:\temp\foo.xml`)
	h.Step()

	h.Resize(900, 100)
	frame := h.Step()

	ops := frame.OpsOfKind(guitest.OpDrawText)
	if len(ops) != 1 {
		t.Fatalf("got %d text ops, want 1: %v", len(ops), frame.Texts())
	}
	if got := ops[0].A.X; got < field.Bounds().X {
		t.Errorf("text drawn at x=%v, left of the field at x=%v: the scroll offset was not released",
			got, field.Bounds().X)
	}
}

// Narrowing still scrolls to keep the caret in view.
func TestTextStaysScrolledWhileItOverflows(t *testing.T) {
	h := guitest.New(900, 100)

	field := ui.NewTextField()
	root := ui.NewContainer()
	root.SetLayout(ui.VBox(0))
	root.Add(field, ui.Align(geom.AlignStretch))
	h.SetContent(root)

	h.Step()
	field.SetText(`c:\a\very\long\path\that\does\not\fit\at\all\foo.xml`)
	h.Step()

	h.Resize(80, 100)
	frame := h.Step()

	ops := frame.OpsOfKind(guitest.OpDrawText)
	if len(ops) != 1 {
		t.Fatalf("got %d text ops, want 1: %v", len(ops), frame.Texts())
	}
	if got := ops[0].A.X; got >= field.Bounds().X {
		t.Errorf("text drawn at x=%v with the caret at the end of overflowing text; it should be scrolled left of the field at x=%v",
			got, field.Bounds().X)
	}
}
