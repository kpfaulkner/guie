package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

func shiftKey(w Widget, k render.Key) {
	ev := Event{Type: EventKeyDown, Key: k, Modifiers: render.ModifierSet(render.ModShift)}
	w.HandleEvent(&ev)
}

func ctrlKey(w Widget, k render.Key) {
	ev := Event{Type: EventKeyDown, Key: k, Modifiers: render.ModifierSet(render.ModControl)}
	w.HandleEvent(&ev)
}

// --- TextField selection ---

func TestTextFieldShiftSelectsAndTypingReplaces(t *testing.T) {
	tf := NewTextField()
	tf.SetText("hello") // caret at end (5), no selection

	shiftKey(tf, render.KeyLeft)
	shiftKey(tf, render.KeyLeft) // select "lo"
	if !tf.hasSelection() {
		t.Fatal("Shift+Left should create a selection")
	}
	lo, hi := tf.selRange()
	if lo != 3 || hi != 5 {
		t.Fatalf("selection should be [3,5), got [%d,%d)", lo, hi)
	}

	typeRune(tf, 'p') // replaces "lo"
	if tf.Text() != "help" {
		t.Fatalf("typing should replace the selection: got %q", tf.Text())
	}
	if tf.hasSelection() {
		t.Fatal("selection should be collapsed after typing")
	}
}

func TestTextFieldSelectAllAndDelete(t *testing.T) {
	tf := NewTextField()
	tf.SetText("erase me")
	ctrlKey(tf, render.KeyA)
	lo, hi := tf.selRange()
	if lo != 0 || hi != len([]rune("erase me")) {
		t.Fatalf("Ctrl+A should select all, got [%d,%d)", lo, hi)
	}
	keyDown(tf, render.KeyBackspace)
	if tf.Text() != "" {
		t.Fatalf("Backspace over a full selection should clear the text, got %q", tf.Text())
	}
}

func TestTextFieldClickAndDragSelects(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)
	tf.SetText("abcdef")
	tf.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 30})
	f := tf.face()

	// Press at index 1, drag to index 4 → selection [1,4).
	x1 := tf.Bounds().X + textFieldPadding.Left + f.Measure("a").W
	x4 := tf.Bounds().X + textFieldPadding.Left + f.Measure("abcd").W
	tf.HandleEvent(&Event{Type: EventPointerDown, Pos: geom.Point{X: x1, Y: 10}})
	tf.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: x4, Y: 10}})

	lo, hi := tf.selRange()
	if lo != 1 || hi != 4 {
		t.Fatalf("click+drag should select [1,4), got [%d,%d)", lo, hi)
	}
}

// --- TextArea selection ---

func TestTextAreaShiftSelectAcrossLinesAndReplace(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("ab\ncd")
	ta.caretRow, ta.caretCol = 1, 0 // start of the second line
	ta.collapse()

	// Shift+Left extends the selection back across the line boundary to (0,2),
	// selecting the newline between the two lines.
	shiftKey(ta, render.KeyLeft)
	if !ta.hasSelection() {
		t.Fatal("Shift+Left should create a selection")
	}
	typeRune(ta, 'X') // replaces the selected range, joining the lines
	if ta.Text() != "abXcd" {
		t.Fatalf("typing should replace a multi-line selection: got %q", ta.Text())
	}
}

func TestTextAreaSelectAll(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("ab\ncd\nef")
	ctrlKey(ta, render.KeyA)
	sr, sc, er, ec := ta.selRange()
	if sr != 0 || sc != 0 || er != 2 || ec != 2 {
		t.Fatalf("Ctrl+A should select all, got (%d,%d)..(%d,%d)", sr, sc, er, ec)
	}
	keyDown(ta, render.KeyDelete)
	if ta.Text() != "" {
		t.Fatalf("Delete over a full selection should clear all text, got %q", ta.Text())
	}
}
