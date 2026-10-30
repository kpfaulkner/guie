package guitest_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/ui"
)

func TestFileDialogMultipleKeyboardOnlyNonContiguous(t *testing.T) {
	e := openMulti(t)
	e.h.TypeKey(render.KeyDown) // docs/
	e.h.TypeKey(render.KeyDown) // notes.txt, selected
	e.wantDrawn("notes.txt")
	e.keyWith(fdPrimary, render.KeyDown) // cursor to pic.png, nothing selected
	e.keyWith(fdPrimary, render.KeyDown) // cursor to readme.md
	e.wantDrawn("notes.txt")             // the name field still shows only notes.txt
	e.h.TypeKey(render.KeySpace)         // select readme.md too
	e.wantDrawn(`"notes.txt" "readme.md"`)
	e.h.TypeKey(render.KeyEnter) // the cursor row is part of the selection: take both
	e.wantResult(true, "/notes.txt", "/readme.md")
}

func TestFileDialogMultipleSpaceDeselects(t *testing.T) {
	e := openMulti(t)
	e.click("notes.txt")
	e.clickWith(fdPrimary, "pic.png")
	e.wantDrawn(`"notes.txt" "pic.png"`)
	e.h.TypeKey(render.KeySpace) // the cursor is on pic.png: deselect it
	e.click("Open")
	e.wantResult(true, "/notes.txt")
}

func TestFileDialogMultipleSpaceDoesNotBreakTypeAhead(t *testing.T) {
	e := openMulti(t)
	e.h.TypeKey(render.KeyDown) // docs/
	e.h.TypeKey(render.KeySpace)
	e.h.TypeText(" ") // the character a real Space also produces
	e.h.Step()
	e.h.TypeText("p")
	e.h.Step() // a prefix of "p", not " p"
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

func TestFileDialogTypeAheadSpaceContinuesAPrefix(t *testing.T) {
	fsys := fstest.MapFS{
		"my file.txt": {Data: []byte("1")},
		"my folder/x": {Data: []byte("1")},
		"mz":          {Data: []byte("1")},
	}
	e := openFD(t, fsys)
	for _, r := range "my fi" {
		e.h.TypeText(string(r))
		e.h.Step()
	}
	e.h.TypeKey(render.KeyEnter)
	e.wantResult(true, "/my file.txt")
}

func TestFileDialogTypeAheadIgnoresALeadingSpace(t *testing.T) {
	e := openFD(t, fdFS())
	e.h.TypeText(" ")
	e.h.Step()
	e.h.TypeText("p")
	e.h.Step()
	e.click("Open")
	e.wantResult(true, "/pic.png")
}

// TestTableCursorOutlineIsDrawn checks the keyboard cursor is visible when it is
// on a row that is not selected, which is the one case the selection highlight
// cannot show it.
func TestTableCursorOutlineIsDrawn(t *testing.T) {
	h := guitest.New(300, 200)
	tbl := ui.NewTable([]ui.Column{{Title: "Name"}})
	rows := make([][]string, 5)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("row%d", i)}
	}
	tbl.SetRows(rows)
	tbl.SetMultiSelect(true)
	h.SetContent(tbl)
	h.Step()

	strokes := func() int { return len(h.Frame().OpsOfKind(guitest.OpStrokeRect)) }
	textPoint := func(s string) geom.Point {
		for _, op := range h.Frame().OpsOfKind(guitest.OpDrawText) {
			if op.Text == s {
				return geom.Point{X: op.A.X + 3, Y: op.A.Y + 3}
			}
		}
		t.Fatalf("%q not drawn", s)
		return geom.Point{}
	}

	p := textPoint("row1")
	h.Click(p.X, p.Y) // selects row 1 and focuses the table
	base := strokes()

	h.SetModifiers(fdPrimary)
	h.TypeKey(render.KeyDown) // cursor to row 2, which is not selected
	h.SetModifiers(0)
	if got := strokes(); got != base+1 {
		t.Errorf("cursor on an unselected row: %d strokes, want %d (one more)", got, base+1)
	}

	h.TypeKey(render.KeySpace) // select it: the highlight shows it, no outline needed
	if got := strokes(); got != base {
		t.Errorf("cursor on a selected row: %d strokes, want %d", got, base)
	}

	h.TypeKey(render.KeySpace) // deselect again: the outline comes back
	if got := strokes(); got != base+1 {
		t.Errorf("after deselecting: %d strokes, want %d", got, base+1)
	}
}

func TestTableNoCursorOutlineWithoutMultiSelect(t *testing.T) {
	h := guitest.New(300, 200)
	tbl := ui.NewTable([]ui.Column{{Title: "Name"}})
	tbl.SetRows([][]string{{"row0"}, {"row1"}})
	h.SetContent(tbl)
	h.Step()
	base := len(h.Frame().OpsOfKind(guitest.OpStrokeRect))
	h.TypeKey(render.KeyDown)
	if got := len(h.Frame().OpsOfKind(guitest.OpStrokeRect)); got != base {
		t.Errorf("single-select table drew an outline: %d strokes, want %d", got, base)
	}
}
