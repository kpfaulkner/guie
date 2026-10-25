package ui

import (
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/render"
)

// textWidget is the surface the undo tests drive on both TextField and TextArea.
type textWidget interface {
	Widget
	Text() string
	SetText(string)
	Undo() bool
	Redo() bool
	CanUndo() bool
	CanRedo() bool
}

func undoType(w Widget, s string) {
	for _, r := range s {
		w.HandleEvent(&Event{Type: EventText, Rune: r})
	}
}

func undoKey(w Widget, k render.Key, mods ...render.Modifier) {
	var ms render.ModifierSet
	for _, m := range mods {
		ms |= render.ModifierSet(m)
	}
	w.HandleEvent(&Event{Type: EventKeyDown, Key: k, Modifiers: ms})
}

func undoWant(t *testing.T, w textWidget, want string) {
	t.Helper()
	if got := w.Text(); got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

// both runs fn against a fresh TextField and a fresh TextArea.
func undoBoth(t *testing.T, fn func(t *testing.T, w textWidget)) {
	t.Run("field", func(t *testing.T) { fn(t, NewTextField()) })
	t.Run("area", func(t *testing.T) { fn(t, NewTextArea()) })
}

func TestUndoRedoTyping(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		var changes []string
		switch x := w.(type) {
		case *TextField:
			x.OnChange(func(s string) { changes = append(changes, s) })
		case *TextArea:
			x.OnChange(func(s string) { changes = append(changes, s) })
		}
		undoType(w, "abc")
		if !w.CanUndo() || w.CanRedo() {
			t.Fatal("after typing: CanUndo should be true, CanRedo false")
		}
		if !w.Undo() {
			t.Fatal("Undo reported nothing to undo")
		}
		undoWant(t, w, "")
		if !w.CanRedo() {
			t.Fatal("CanRedo false after Undo")
		}
		if !w.Redo() {
			t.Fatal("Redo reported nothing to redo")
		}
		undoWant(t, w, "abc")
		w.Undo()
		if w.Undo() {
			t.Fatal("second Undo should report false")
		}
		if last := changes[len(changes)-1]; last != "" {
			t.Errorf("OnChange last = %q, want empty after undo", last)
		}
	})
}

func TestUndoSplitsAtWordBoundary(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "hello world")
		w.Undo()
		undoWant(t, w, "hello")
		w.Undo()
		undoWant(t, w, "")
	})
}

func TestUndoKeys(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "abc")
		undoKey(w, render.KeyZ, render.ModPrimary)
		undoWant(t, w, "")
		undoKey(w, render.KeyZ, render.ModPrimary, render.ModShift)
		undoWant(t, w, "abc")
		undoKey(w, render.KeyZ, render.ModPrimary)
		undoKey(w, render.KeyY, render.ModPrimary)
		undoWant(t, w, "abc")
		undoKey(w, render.KeyZ) // no modifier: not an undo
		undoWant(t, w, "abc")
	})
}

func TestUndoBackspaceCoalesces(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		w.SetText("abcdef")
		undoKey(w, render.KeyBackspace)
		undoKey(w, render.KeyBackspace)
		undoKey(w, render.KeyBackspace)
		undoWant(t, w, "abc")
		w.Undo()
		undoWant(t, w, "abcdef")
		if w.CanUndo() {
			t.Fatal("three backspaces should have been one step")
		}
	})
}

func TestUndoForwardDeleteCoalesces(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		w.SetText("abcdef")
		undoKey(w, render.KeyHome)
		undoKey(w, render.KeyDelete)
		undoKey(w, render.KeyDelete)
		undoWant(t, w, "cdef")
		w.Undo()
		undoWant(t, w, "abcdef")
		if w.CanUndo() {
			t.Fatal("two forward deletes should have been one step")
		}
	})
}

func TestUndoCaretMoveSplitsSteps(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "ab")
		undoKey(w, render.KeyLeft)
		undoType(w, "X")
		undoWant(t, w, "aXb")
		w.Undo()
		undoWant(t, w, "ab")
		w.Undo()
		undoWant(t, w, "")
	})
}

func TestUndoReplaceSelectionIsOneStepAndRestoresSelection(t *testing.T) {
	t.Run("field", func(t *testing.T) {
		f := NewTextField()
		f.SetText("hello world")
		f.anchor, f.caret = 0, 5 // "hello" selected
		f.insertString("bye")
		undoWant(t, f, "bye world")
		f.Undo()
		undoWant(t, f, "hello world")
		if lo, hi := f.selRange(); lo != 0 || hi != 5 {
			t.Errorf("selection after undo = %d..%d, want 0..5", lo, hi)
		}
		f.Redo()
		undoWant(t, f, "bye world")
		if f.hasSelection() || f.caret != 3 {
			t.Errorf("after redo caret=%d sel=%v, want caret 3 no selection", f.caret, f.hasSelection())
		}
	})
	t.Run("area", func(t *testing.T) {
		a := NewTextArea()
		a.SetText("hello\nworld")
		a.SelectRange(2, 8) // "llo\nwo"
		a.insertString("X")
		undoWant(t, a, "heXrld")
		a.Undo()
		undoWant(t, a, "hello\nworld")
		if s, e := a.SelectionRange(); s != 2 || e != 8 {
			t.Errorf("selection after undo = %d..%d, want 2..8", s, e)
		}
		a.Redo()
		undoWant(t, a, "heXrld")
	})
}

func TestUndoCutRestoresTextAndSelection(t *testing.T) {
	f := NewTextField()
	f.SetText("abcdef")
	f.anchor, f.caret = 1, 4
	f.cutSelection()
	undoWant(t, f, "aef")
	f.Undo()
	undoWant(t, f, "abcdef")
	if lo, hi := f.selRange(); lo != 1 || hi != 4 {
		t.Errorf("selection = %d..%d, want 1..4", lo, hi)
	}
}

func TestUndoNewEditClearsRedo(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "ab")
		w.Undo()
		undoType(w, "z")
		if w.CanRedo() {
			t.Fatal("a new edit should clear redo")
		}
		undoWant(t, w, "z")
	})
}

func TestUndoNotMergedAcrossUndo(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "ab cd")
		w.Undo() // leaves "ab" with the caret exactly where that step ended
		undoType(w, "x")
		undoWant(t, w, "abx")
		w.Undo()
		undoWant(t, w, "ab") // x was its own step, not folded into "ab"
	})
}

func TestUndoSetTextClearsHistory(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "abc")
		w.Undo()
		w.SetText("loaded")
		if w.CanUndo() || w.CanRedo() {
			t.Fatal("SetText should clear history")
		}
	})
}

func TestUndoFocusLossSeals(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "ab")
		w.HandleEvent(&Event{Type: EventFocusLost})
		w.HandleEvent(&Event{Type: EventFocusGained})
		undoType(w, "cd")
		undoWant(t, w, "abcd")
		w.Undo()
		undoWant(t, w, "ab")
	})
}

func TestUndoDepthCapped(t *testing.T) {
	f := NewTextField()
	for i := 0; i < undoDepth+10; i++ {
		f.insertString("x") // kindOther: one step each
	}
	n := 0
	for f.Undo() {
		n++
	}
	if n != undoDepth {
		t.Errorf("undoable steps = %d, want %d", n, undoDepth)
	}
	if got := len(f.Text()); got != 10 {
		t.Errorf("text left after undoing all = %d runes, want 10", got)
	}
}

func TestUndoDisabledIgnoresKeys(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "abc")
		type enabler interface{ SetEnabled(bool) }
		w.(enabler).SetEnabled(false)
		undoKey(w, render.KeyZ, render.ModPrimary)
		undoWant(t, w, "abc")
	})
}

func TestUndoMaskedFieldKeepsRealText(t *testing.T) {
	f := NewTextField(Masked())
	undoType(f, "secret")
	f.Undo()
	undoWant(t, f, "")
	f.Redo()
	undoWant(t, f, "secret")
}

func TestUndoClearHistory(t *testing.T) {
	undoBoth(t, func(t *testing.T, w textWidget) {
		undoType(w, "abc")
		type clearer interface{ ClearHistory() }
		w.(clearer).ClearHistory()
		if w.CanUndo() {
			t.Fatal("ClearHistory should leave nothing to undo")
		}
		undoWant(t, w, "abc")
	})
}

func TestUndoAreaMultiLine(t *testing.T) {
	a := NewTextArea()
	undoType(a, "ab")
	undoKey(a, render.KeyEnter)
	undoType(a, "cd")
	undoWant(t, a, "ab\ncd")

	// Backspace at the start of a line joins it with the previous one.
	undoKey(a, render.KeyHome)
	undoKey(a, render.KeyBackspace)
	undoWant(t, a, "abcd")
	a.Undo()
	undoWant(t, a, "ab\ncd")
	if a.caretRow != 1 || a.caretCol != 0 {
		t.Errorf("caret after undoing the join = (%d,%d), want (1,0)", a.caretRow, a.caretCol)
	}

	a.insertString("1\n2\n3")
	undoWant(t, a, "ab\n1\n2\n3cd")
	a.Undo()
	undoWant(t, a, "ab\ncd")
	a.Redo()
	undoWant(t, a, "ab\n1\n2\n3cd")

	// Delete a selection spanning several lines, then bring it back.
	a.SelectRange(1, 7)
	want := a.Text()
	a.cutSelection()
	a.Undo()
	undoWant(t, a, want)
	if s, e := a.SelectionRange(); s != 1 || e != 7 {
		t.Errorf("selection after undo = %d..%d, want 1..7", s, e)
	}

	// Unwind everything.
	for a.Undo() {
	}
	undoWant(t, a, "")
}

func TestUndoAreaTypingTouchesOnlyItsLine(t *testing.T) {
	a := NewTextArea()
	a.SetText(strings.Repeat("line\n", 5000) + "end")
	before := a.lines
	first := &a.lines[0][0]
	undoType(a, "x")
	if len(a.lines) != len(before) || &a.lines[0] != &before[0] {
		t.Error("typing within a line rebuilt the line list")
	}
	if &a.lines[0][0] != first {
		t.Error("typing touched an unrelated line")
	}
	a.Undo()
	undoWant(t, a, strings.Repeat("line\n", 5000)+"end")
}

func TestReplaceRangeIsOneUndoableStep(t *testing.T) {
	t.Run("field", func(t *testing.T) {
		f := NewTextField()
		f.SetText("one two one")
		var got string
		f.OnChange(func(s string) { got = s })
		f.ReplaceRange(0, len("one two one"), "1 two 1")
		undoWant(t, f, "1 two 1")
		if got != "1 two 1" || f.caret != 7 || f.hasSelection() {
			t.Errorf("OnChange=%q caret=%d sel=%v", got, f.caret, f.hasSelection())
		}
		f.Undo()
		undoWant(t, f, "one two one")
		f.Redo()
		undoWant(t, f, "1 two 1")
	})
	t.Run("area", func(t *testing.T) {
		a := NewTextArea()
		a.SetText("a\nb\na")
		a.ReplaceRange(0, 5, "x\ny\nx")
		undoWant(t, a, "x\ny\nx")
		if a.CanUndo() != true {
			t.Fatal("ReplaceRange not recorded")
		}
		a.Undo()
		undoWant(t, a, "a\nb\na")
		if a.CanUndo() {
			t.Fatal("ReplaceRange should be exactly one step")
		}
		a.Redo()
		undoWant(t, a, "x\ny\nx")
	})
}

func TestReplaceRangeClampsAndOrders(t *testing.T) {
	f := NewTextField()
	f.SetText("abcdef")
	f.ReplaceRange(100, 4, "X") // reversed and out of range: replaces "ef"
	undoWant(t, f, "abcdX")
	a := NewTextArea()
	a.SetText("abc\ndef")
	a.ReplaceRange(-5, 2, "Z")
	undoWant(t, a, "Zc\ndef")
	a.ReplaceRange(2, 99, "")
	undoWant(t, a, "Zc")
}

func TestReplaceRangeDoesNotMergeWithTyping(t *testing.T) {
	both := func(w textWidget, rr func(int, int, string)) {
		undoType(w, "ab")
		rr(2, 2, "X") // insert at the caret, like the next typed rune
		undoType(w, "c")
		w.Undo()
		undoWant(t, w, "abX")
		w.Undo()
		undoWant(t, w, "ab")
	}
	f, a := NewTextField(), NewTextArea()
	both(f, f.ReplaceRange)
	both(a, a.ReplaceRange)
}
