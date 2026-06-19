package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/render"
)

func TestWrapOffIsOneRowPerLine(t *testing.T) {
	ta := NewTextArea()
	NewApp().SetContent(ta)
	ta.SetText("a b c d e f g h")
	ta.wrapWidth = 5 // would wrap, but wrap is disabled
	if got := len(ta.rows()); got != 1 {
		t.Fatalf("with wrap off there should be one visual row per line, got %d", got)
	}
}

func TestWrapSplitsLongLine(t *testing.T) {
	ta := NewTextArea(TextAreaWrap())
	NewApp().SetContent(ta)
	f := ta.face()
	ta.SetText("alpha beta gamma delta")
	// A width that holds roughly two words forces multiple segments.
	ta.wrapWidth = f.Measure("alpha beta ").W

	segs := ta.wrapSegments(ta.lines[0])
	if len(segs) < 2 {
		t.Fatalf("expected the long line to wrap into multiple segments, got %d", len(segs))
	}
	// Segments must cover the line contiguously from 0 to len.
	if segs[0][0] != 0 {
		t.Fatalf("first segment should start at 0, got %d", segs[0][0])
	}
	for i := 1; i < len(segs); i++ {
		if segs[i][0] != segs[i-1][1] {
			t.Fatalf("segments must be contiguous: seg %d starts at %d, prev ends at %d", i, segs[i][0], segs[i-1][1])
		}
	}
	if last := segs[len(segs)-1][1]; last != len(ta.lines[0]) {
		t.Fatalf("last segment should end at line end %d, got %d", len(ta.lines[0]), last)
	}

	// rows() over a single logical line equals the segment count.
	if len(ta.rows()) != len(segs) {
		t.Fatalf("rows() should match segment count for one logical line")
	}
}

func TestWrapVerticalMoveStaysInLogicalLine(t *testing.T) {
	ta := NewTextArea(TextAreaWrap())
	NewApp().SetContent(ta)
	f := ta.face()
	ta.SetText("alpha beta gamma delta")
	ta.wrapWidth = f.Measure("alpha beta ").W

	// Caret at the start; Down should move to the next visual row of the SAME
	// logical line (row stays 0), not to a different logical line.
	ta.caretRow, ta.caretCol = 0, 0
	ta.collapse()
	rows := ta.rows()
	if len(rows) < 2 {
		t.Skip("line did not wrap; font metrics too narrow")
	}
	keyDown(ta, render.KeyDown)
	if ta.caretRow != 0 {
		t.Fatalf("Down within a wrapped line should keep the logical row at 0, got %d", ta.caretRow)
	}
	if ta.caretCol == 0 {
		t.Fatalf("Down should advance the caret into the next visual row")
	}
}
