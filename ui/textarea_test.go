package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/render"
)

func typeStr(t *TextArea, s string) {
	for _, r := range s {
		ev := Event{Type: EventText, Rune: r}
		t.HandleEvent(&ev)
	}
}

func taKey(t *TextArea, k render.Key) {
	ev := Event{Type: EventKeyDown, Key: k}
	t.HandleEvent(&ev)
}

func TestTextAreaTypingAndNewline(t *testing.T) {
	ta := NewTextArea()
	typeStr(ta, "ab")
	taKey(ta, render.KeyEnter)
	typeStr(ta, "cd")

	if ta.Text() != "ab\ncd" {
		t.Fatalf("got %q, want %q", ta.Text(), "ab\ncd")
	}
	if ta.caretRow != 1 || ta.caretCol != 2 {
		t.Fatalf("caret at row %d col %d, want 1,2", ta.caretRow, ta.caretCol)
	}
}

func TestTextAreaNewlineSplitsMidLine(t *testing.T) {
	ta := NewTextArea()
	typeStr(ta, "abcd")
	taKey(ta, render.KeyLeft)
	taKey(ta, render.KeyLeft) // caret between b and c
	taKey(ta, render.KeyEnter)

	if ta.Text() != "ab\ncd" {
		t.Fatalf("splitting mid-line: got %q want %q", ta.Text(), "ab\ncd")
	}
}

func TestTextAreaBackspaceJoinsLines(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("ab\ncd")
	// caret is at end of "cd" (row 1, col 2). Move home, then backspace joins.
	taKey(ta, render.KeyHome)
	taKey(ta, render.KeyBackspace)

	if ta.Text() != "abcd" {
		t.Fatalf("backspace at line start should join: got %q", ta.Text())
	}
	if ta.caretRow != 0 || ta.caretCol != 2 {
		t.Fatalf("caret should be at the join point (0,2), got %d,%d", ta.caretRow, ta.caretCol)
	}
}

func TestTextAreaDeleteJoinsLines(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("ab\ncd")
	ta.caretRow, ta.caretCol = 0, 2 // end of first line
	ta.collapse()                   // no selection
	taKey(ta, render.KeyDelete)
	if ta.Text() != "abcd" {
		t.Fatalf("delete at line end should join: got %q", ta.Text())
	}
}

func TestTextAreaVerticalCaretMovement(t *testing.T) {
	ta := NewTextArea()
	NewApp().SetContent(ta) // mount so the font is available for x-preserving moves
	ta.SetText("long line\nx")
	ta.caretRow, ta.caretCol = 0, 9 // end of "long line"
	ta.collapse()                   // no selection
	taKey(ta, render.KeyDown)       // move to short line; col clamps
	if ta.caretRow != 1 || ta.caretCol != 1 {
		t.Fatalf("Down should clamp column to short line: got %d,%d", ta.caretRow, ta.caretCol)
	}
	taKey(ta, render.KeyUp)
	if ta.caretRow != 0 {
		t.Fatalf("Up should return to first line, got row %d", ta.caretRow)
	}
}

func TestTextAreaArrowsAcrossLineBoundary(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("ab\ncd")
	ta.caretRow, ta.caretCol = 1, 0 // start of second line
	ta.collapse()                   // no selection
	taKey(ta, render.KeyLeft)       // should wrap to end of first line
	if ta.caretRow != 0 || ta.caretCol != 2 {
		t.Fatalf("Left at line start should move to prev line end, got %d,%d", ta.caretRow, ta.caretCol)
	}
	taKey(ta, render.KeyRight) // back to start of second line
	if ta.caretRow != 1 || ta.caretCol != 0 {
		t.Fatalf("Right at line end should move to next line start, got %d,%d", ta.caretRow, ta.caretCol)
	}
}

func TestTextAreaOnChange(t *testing.T) {
	got := ""
	ta := NewTextArea(OnTextAreaChange(func(s string) { got = s }))
	typeStr(ta, "hi")
	if got != "hi" {
		t.Fatalf("OnTextAreaChange should report %q, got %q", "hi", got)
	}
}
