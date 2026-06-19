package ui

import "testing"

func TestTextAreaOffsetRoundTrip(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("ab\ncd\nef") // offsets: a0 b1 \n2 c3 d4 \n5 e6 f7
	cases := []struct {
		row, col, off int
	}{
		{0, 0, 0}, {0, 2, 2}, {1, 0, 3}, {1, 1, 4}, {2, 2, 8},
	}
	for _, c := range cases {
		if got := ta.posToOffset(c.row, c.col); got != c.off {
			t.Errorf("posToOffset(%d,%d)=%d want %d", c.row, c.col, got, c.off)
		}
		r, col := ta.offsetToPos(c.off)
		if r != c.row || col != c.col {
			t.Errorf("offsetToPos(%d)=(%d,%d) want (%d,%d)", c.off, r, col, c.row, c.col)
		}
	}
}

func TestTextAreaCaretOffsetAfterSetText(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("hello")
	if ta.CaretOffset() != 5 {
		t.Fatalf("caret should be at end (5), got %d", ta.CaretOffset())
	}
}

func TestTextAreaSelectRange(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("abcdef")
	ta.SelectRange(1, 4)
	s, e := ta.SelectionRange()
	if s != 1 || e != 4 {
		t.Fatalf("SelectionRange = (%d,%d) want (1,4)", s, e)
	}
	if ta.CaretOffset() != 4 {
		t.Fatalf("caret should sit at the selection end (4), got %d", ta.CaretOffset())
	}
	// Order-independent.
	ta.SelectRange(5, 2)
	if s, e := ta.SelectionRange(); s != 2 || e != 5 {
		t.Fatalf("reversed SelectRange = (%d,%d) want (2,5)", s, e)
	}
}

func TestTextAreaFind(t *testing.T) {
	ta := NewTextArea()
	ta.SetText("the cat sat on the mat")

	idx, ok := ta.Find("cat", 0)
	if !ok || idx != 4 {
		t.Fatalf("Find cat from 0: idx=%d ok=%v", idx, ok)
	}
	if s, e := ta.SelectionRange(); s != 4 || e != 7 {
		t.Fatalf("Find should select the match, got (%d,%d)", s, e)
	}

	// Searching past the match misses; "the" appears again later.
	if _, ok := ta.Find("cat", 5); ok {
		t.Fatal("Find cat from 5 should miss")
	}
	if idx, ok := ta.Find("the", 1); !ok || idx != 15 {
		t.Fatalf("Find the from 1 should find the second 'the' at 15, got idx=%d ok=%v", idx, ok)
	}
	if _, ok := ta.Find("zzz", 0); ok {
		t.Fatal("Find of absent text should miss")
	}
}

func TestModalFocusConfinedToModal(t *testing.T) {
	app := NewApp()
	rootBtn := NewButton("root")
	app.SetContent(rootBtn)

	panel := NewContainer()
	panel.SetLayout(VBox(0))
	a := NewButton("a")
	b := NewButton("b")
	panel.Add(a)
	panel.Add(b)
	app.ShowModal(panel)

	app.moveFocus(1)
	if app.focused != Widget(a) {
		t.Fatalf("Tab should focus the first widget inside the open modal")
	}
	app.moveFocus(1)
	if app.focused != Widget(b) {
		t.Fatalf("Tab should move to the next widget inside the modal")
	}
	if app.focused == Widget(rootBtn) {
		t.Fatal("focus must not leak to the background behind the modal")
	}
}
