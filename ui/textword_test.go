package ui

import (
	"testing"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

func TestWordBounds(t *testing.T) {
	cases := []struct {
		text    string
		idx     int
		lo, hi  int
		comment string
	}{
		{"hello world", 2, 0, 5, "inside first word"},
		{"hello world", 8, 6, 11, "inside second word"},
		{"hello world", 5, 0, 5, "just past a word selects that word"},
		{"hello world", 6, 6, 11, "start of a word"},
		{"hello world", 11, 6, 11, "end of text"},
		{"foo_bar9 x", 3, 0, 8, "underscore and digits are word runes"},
		{"a   b", 2, 1, 4, "whitespace run"},
		{"x == y", 3, 2, 4, "punctuation run"},
		{"café ok", 3, 0, 4, "non-ASCII letters"},
		{"", 0, 0, 0, "empty"},
	}
	for _, c := range cases {
		lo, hi := wordBounds([]rune(c.text), c.idx)
		if lo != c.lo || hi != c.hi {
			t.Errorf("%s: wordBounds(%q, %d) = [%d,%d), want [%d,%d)", c.comment, c.text, c.idx, lo, hi, c.lo, c.hi)
		}
	}
}

func TestTextFieldDoubleClickSelectsWord(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)
	tf.SetText("hello world")
	tf.SetBounds(geom.Rect{W: 300, H: 30})
	f := tf.face()
	x := tf.Bounds().X + textFieldPadding.Left + f.Measure("hello wo").W // inside "world"

	tf.HandleEvent(&Event{Type: EventDoubleClick, Button: render.MouseLeft, Pos: geom.Point{X: x, Y: 10}})
	if lo, hi := tf.selRange(); lo != 6 || hi != 11 {
		t.Fatalf("selection = [%d,%d), want [6,11)", lo, hi)
	}
	if tf.selectedText() != "world" {
		t.Errorf("selected %q, want %q", tf.selectedText(), "world")
	}
}

func TestTextFieldDoubleClickIgnoresRightButton(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	app.SetContent(tf)
	tf.SetText("hello world")
	tf.SetBounds(geom.Rect{W: 300, H: 30})
	tf.HandleEvent(&Event{Type: EventDoubleClick, Button: render.MouseRight, Pos: geom.Point{X: 20, Y: 10}})
	if tf.hasSelection() {
		t.Error("only a left double-click selects a word")
	}
}

func TestMaskedFieldDoubleClickSelectsAllNotWord(t *testing.T) {
	app := NewApp()
	tf := NewTextField(Masked())
	app.SetContent(tf)
	tf.SetText("my secret key")
	tf.SetBounds(geom.Rect{W: 300, H: 30})
	tf.HandleEvent(&Event{Type: EventDoubleClick, Button: render.MouseLeft, Pos: geom.Point{X: 30, Y: 10}})
	if lo, hi := tf.selRange(); lo != 0 || hi != 13 {
		t.Errorf("masked selection = [%d,%d), want all [0,13): word edges would reveal the spaces", lo, hi)
	}
}

func TestTextAreaDoubleClickSelectsWordOnItsLine(t *testing.T) {
	app := NewApp()
	ta := NewTextArea()
	app.SetContent(ta)
	ta.SetText("one two\nthree four")
	ta.SetBounds(geom.Rect{W: 300, H: 100})
	f := ta.face()
	inner := ta.Bounds().Inset(textAreaPadding)
	x := inner.X + f.Measure("three fo").W // inside "four"
	y := inner.Y + ta.lineHeight()*1.5     // second line

	ta.HandleEvent(&Event{Type: EventDoubleClick, Button: render.MouseLeft, Pos: geom.Point{X: x, Y: y}})
	if ta.selectedText() != "four" {
		t.Fatalf("selected %q, want %q", ta.selectedText(), "four")
	}
}

func TestDoubleClickSelectsWordEndToEnd(t *testing.T) {
	app := NewApp()
	tf := NewTextField()
	root := NewContainer()
	root.SetBounds(geom.Rect{W: 300, H: 100})
	tf.SetBounds(geom.Rect{X: 10, Y: 10, W: 200, H: 30})
	root.Add(tf)
	app.SetContent(root)
	tf.SetText("alpha beta")
	now := time.Unix(1000, 0)
	app.now = func() time.Time { return now }

	f := tf.face()
	pos := geom.Point{X: tf.Bounds().X + textFieldPadding.Left + f.Measure("alpha b").W, Y: 20}
	dcClick(app, pos)
	now = now.Add(80 * time.Millisecond)
	dcClick(app, pos)
	if tf.selectedText() != "beta" {
		t.Errorf("selected %q after double-click, want %q", tf.selectedText(), "beta")
	}
}
