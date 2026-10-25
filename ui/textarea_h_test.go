package ui

import (
	"strings"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// hArea returns a TextArea of the given size holding text, caret at the start.
func hArea(t *testing.T, w, h float64, text string, opts ...TextAreaOption) *TextArea {
	t.Helper()
	app := NewApp()
	ta := NewTextArea(opts...)
	app.SetContent(ta)
	ta.SetBounds(geom.Rect{X: 10, Y: 20, W: w, H: h})
	ta.SetText(text)
	ta.caretRow, ta.caretCol = 0, 0
	ta.collapse()
	return ta
}

var longLine = strings.Repeat("0123456789", 30) // far wider than any test view

func TestTextAreaWrappedHasNoHorizontalBar(t *testing.T) {
	ta := hArea(t, 200, 100, longLine, TextAreaWrap())
	g := ta.geomInfo()
	if g.hOver {
		t.Error("wrapped text never overflows sideways, so there is no horizontal bar")
	}
	if g.view != g.inner {
		t.Errorf("wrapped view %v should be the padded box %v as it always was", g.view, g.inner)
	}
	if ta.maxScrollX() != 0 {
		t.Errorf("maxScrollX = %v, want 0 when wrapping", ta.maxScrollX())
	}
}

func TestTextAreaLongLineShowsHorizontalBar(t *testing.T) {
	ta := hArea(t, 200, 100, longLine)
	g := ta.geomInfo()
	if !g.hOver {
		t.Fatal("a line wider than the view should show the horizontal bar")
	}
	if want := g.inner.H - scrollbarWidth; !near(g.view.H, want) {
		t.Errorf("view height %v, want %v: the bar takes a strip off the text", g.view.H, want)
	}
	if ta.maxScrollX() <= 0 {
		t.Error("there should be somewhere to scroll to")
	}
	if g.vOver {
		t.Error("one line does not overflow vertically")
	}
}

func TestTextAreaShortTextHasNoBars(t *testing.T) {
	ta := hArea(t, 200, 100, "short")
	g := ta.geomInfo()
	if g.hOver || g.vOver {
		t.Errorf("short text: hOver=%v vOver=%v, want neither", g.hOver, g.vOver)
	}
	if g.view != g.inner {
		t.Errorf("view %v, want the padded box %v", g.view, g.inner)
	}
}

func TestTextAreaBothBarsInteract(t *testing.T) {
	// Lines fit vertically only until the horizontal bar takes its strip.
	ta := hArea(t, 200, 100, "x", TextAreaWrap())
	lh := ta.lineHeight()
	n := int(ta.geomInfo().inner.H / lh) // rows that exactly fit
	ta.wrap = false
	lines := make([]string, n)
	lines[0] = longLine
	ta.SetText(strings.Join(lines, "\n"))
	g := ta.geomInfo()
	if !g.hOver || !g.vOver {
		t.Fatalf("hOver=%v vOver=%v, want both: the bar's strip must push a row out", g.hOver, g.vOver)
	}
	v, h := ta.scrollTrack(), ta.scrollTrackH()
	if v.Y+v.H > h.Y+0.001 || h.X+h.W > v.X+0.001 {
		t.Errorf("tracks overlap the corner: vertical %v horizontal %v", v, h)
	}
	if g.view.W > g.inner.W-0 || g.view.X+g.view.W > ta.Bounds().X+ta.Bounds().W-scrollbarWidth+0.001 {
		t.Errorf("text view %v runs under the vertical bar", g.view)
	}
}

func TestTextAreaHorizontalThumbDrag(t *testing.T) {
	ta := hArea(t, 200, 100, longLine)
	tr := ta.scrollTrackH()
	press := geom.Point{X: tr.X + 3, Y: tr.Y + 4}
	ta.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: press})
	ta.HandleEvent(&Event{Type: EventPointerMove, Pos: geom.Point{X: press.X + 40, Y: press.Y - 50}})
	ta.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft, Pos: press})
	if ta.scrollX <= 0 {
		t.Fatalf("dragging the horizontal thumb left scrollX at %v", ta.scrollX)
	}
	if ta.scrollY != 0 || ta.hasSelection() || ta.caretCol != 0 {
		t.Error("a bar drag must not scroll vertically, select, or move the caret")
	}
}

func TestTextAreaWheelAcross(t *testing.T) {
	ta := hArea(t, 200, 100, longLine)
	ta.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{X: -1}})
	if ta.scrollX != wheelStep {
		t.Errorf("sideways wheel: scrollX = %v, want %v", ta.scrollX, float64(wheelStep))
	}
	ta.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if ta.scrollX != 2*wheelStep || ta.scrollY != 0 {
		t.Errorf("shift+wheel: x=%v y=%v, want x=%v y=0", ta.scrollX, ta.scrollY, float64(2*wheelStep))
	}

	// With nothing to scroll across, shift+wheel scrolls vertically as before.
	tall := hArea(t, 200, 100, strings.Repeat("ab\n", 60))
	tall.HandleEvent(&Event{Type: EventWheel, Wheel: geom.Point{Y: -1}, Modifiers: render.ModifierSet(render.ModShift)})
	if tall.scrollY != wheelStep {
		t.Errorf("no range across: scrollY = %v, want %v", tall.scrollY, float64(wheelStep))
	}
}

func TestTextAreaClickMapsThroughHorizontalScroll(t *testing.T) {
	ta := hArea(t, 200, 100, longLine)
	f := ta.face()
	inner := ta.geomInfo().inner
	y := inner.Y + ta.lineHeight()/2

	// Unscrolled, a click 5 characters in lands at column 5.
	ta.HandleEvent(&Event{Type: EventPointerDown, Button: render.MouseLeft, Pos: geom.Point{X: inner.X + f.Measure(longLine[:5]).W, Y: y}})
	ta.HandleEvent(&Event{Type: EventPointerUp, Button: render.MouseLeft})
	if ta.caretCol != 5 {
		t.Fatalf("caret col = %d, want 5", ta.caretCol)
	}

	// Scrolled 100px across, the same screen x is further along the line.
	ta.scrollX = 100
	at := inner.X + f.Measure(longLine[:5]).W
	r, c := ta.caretAt(geom.Point{X: at, Y: y})
	if r != 0 || c <= 5 {
		t.Errorf("scrolled click: row %d col %d, want a column past 5", r, c)
	}
	want := caretIndexForX([]rune(longLine), at-inner.X+100, f)
	if c != want {
		t.Errorf("scrolled click col = %d, want %d", c, want)
	}

	// The IME candidate window follows the scroll too.
	ta.focused = true
	ta.caretCol, ta.scrollX = 40, 0
	r0, _ := ta.imeCaretRect()
	ta.scrollX = 100
	r1, _ := ta.imeCaretRect()
	if !near(r0.X-r1.X, 100) {
		t.Errorf("IME caret x moved by %v, want 100 when scrolled across", r0.X-r1.X)
	}
}

func TestTextAreaFollowsCaretAcross(t *testing.T) {
	ta := hArea(t, 200, 100, longLine)
	ta.focused = true
	ta.caretRow, ta.caretCol = 0, len(longLine) // the end of the long line
	ta.collapse()
	ta.scrollCaretIntoView()
	g := ta.geomInfo()
	x := ta.caretXWithPreedit(ta.face(), visRow{lr: 0, start: 0, end: len(longLine)}) - ta.scrollX
	if x < 0 || x > g.view.W {
		t.Errorf("caret at view x=%v, outside the view width %v", x, g.view.W)
	}
	if ta.scrollX <= 0 {
		t.Error("the view should have scrolled across to the end of the line")
	}

	ta.caretCol = 0
	ta.collapse()
	ta.scrollCaretIntoView()
	if ta.scrollX != 0 {
		t.Errorf("caret back at the start: scrollX = %v, want 0", ta.scrollX)
	}
}

func TestTextAreaDoesNotSnapBackToTheCaret(t *testing.T) {
	ta := hArea(t, 200, 100, longLine+"\n"+strings.Repeat("ab\n", 60))
	ta.scrollCaretIntoView() // settles on the caret at the start

	// The user scrolls away with the wheel or a bar; the caret has not moved.
	ta.scrollX, ta.scrollY = 120, 200
	ta.scrollCaretIntoView()
	if ta.scrollX != 120 || ta.scrollY != 200 {
		t.Errorf("scroll snapped back to (%v,%v), want (120,200): the caret did not move", ta.scrollX, ta.scrollY)
	}

	// Typing brings the caret back into view (by the least scrolling needed).
	ta.HandleEvent(&Event{Type: EventText, Rune: 'x'})
	ta.scrollCaretIntoView()
	cx := ta.caretXWithPreedit(ta.face(), visRow{lr: 0, start: 0, end: len([]rune(longLine)) + 1}) - ta.scrollX
	if ta.scrollY != 0 || cx < 0 || cx > ta.geomInfo().view.W {
		t.Errorf("after typing scroll = (%v,%v), caret at view x=%v: want it back in view", ta.scrollX, ta.scrollY, cx)
	}
}

func TestTextAreaContentWidthCacheTracksEdits(t *testing.T) {
	ta := hArea(t, 200, 100, "short")
	w0 := ta.contentWidth()
	if w1 := ta.contentWidth(); w1 != w0 {
		t.Errorf("cached width changed with no edit: %v -> %v", w0, w1)
	}
	ta.ReplaceRange(0, 5, longLine)
	if ta.contentWidth() <= w0 {
		t.Error("width should grow after the text gets a long line")
	}
	ta.Undo()
	if got := ta.contentWidth(); !near(got, w0) {
		t.Errorf("width after undo = %v, want back to %v", got, w0)
	}
	ta.SetText(longLine)
	if ta.contentWidth() <= w0 {
		t.Error("SetText should refresh the width")
	}
}

func TestTextAreaSetWrapSwitchesModes(t *testing.T) {
	ta := hArea(t, 200, 100, longLine, TextAreaWrap())
	if ta.geomInfo().hOver {
		t.Fatal("wrapped: no horizontal bar")
	}
	ta.SetWrap(false)
	if !ta.geomInfo().hOver || ta.maxScrollX() <= 0 {
		t.Error("unwrapped, the long line should need the horizontal bar")
	}
	ta.scrollX = 80
	ta.SetWrap(true)
	if ta.scrollX != 0 || ta.geomInfo().hOver {
		t.Errorf("wrapping again: scrollX=%v hOver=%v, want 0 and none", ta.scrollX, ta.geomInfo().hOver)
	}
	ta.SetWrap(true) // no change: must not disturb anything
}
