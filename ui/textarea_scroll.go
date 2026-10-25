package ui

import "github.com/kpfaulkner/guie/geom"

// areaGeom is how a TextArea's box is carved up for text and scrollbars.
type areaGeom struct {
	inner geom.Rect // the box inside its padding
	view  geom.Rect // where text shows: inner, less any scrollbar strips
	vOver bool      // the vertical bar is shown
	hOver bool      // the horizontal bar is shown (only without soft wrap)
}

// geomInfo works out the layout of text and bars together, because each bar's
// strip shortens the other's view: the vertical bar narrows the text, which can
// make a line overflow and need the horizontal bar, whose strip shortens the text
// and can make the rows overflow. The bar only ever takes space, so one extra pass
// settles it.
//
// With soft wrap on, lines never run past the edge, so there is no horizontal bar
// and the view is the padded box as it always was.
func (t *TextArea) geomInfo() areaGeom {
	b := t.Bounds()
	inner := b.Inset(t.insets(textAreaPadding))
	g := areaGeom{inner: inner, view: inner}
	g.vOver = t.contentHeight() > inner.H
	if t.wrap {
		return g
	}
	// The text stops at the vertical bar rather than running under it.
	narrow := func(vOver bool) float64 {
		if !vOver {
			return inner.W
		}
		return minF(inner.W, b.X+b.W-scrollbarWidth-inner.X)
	}
	g.view.W = narrow(g.vOver)
	if t.contentWidth() > g.view.W {
		g.hOver = true
		g.view.H = inner.H - scrollbarWidth
		g.vOver = t.contentHeight() > g.view.H
		g.view.W = narrow(g.vOver)
	}
	return g
}

// textCaretMargin is the room left after the longest line, so a caret at its end
// is not clipped.
const textCaretMargin = 2

// contentWidth is the width of the longest line, plus a caret's room. It is
// cached against the text revision, because measuring every line is the cost of
// the whole document.
func (t *TextArea) contentWidth() float64 {
	if t.widthOK && t.widthRev == t.rev {
		return t.widthCache
	}
	var w float64
	if f := t.face(); f != nil {
		for _, ln := range t.lines {
			w = maxF(w, f.Measure(string(ln)).W)
		}
	}
	t.widthCache, t.widthRev, t.widthOK = w+textCaretMargin, t.rev, true
	return t.widthCache
}

// viewHeight is the height of the area text shows in.
func (t *TextArea) viewHeight() float64 { return t.geomInfo().view.H }

func (t *TextArea) maxScrollX() float64 {
	if t.wrap {
		return 0
	}
	return maxF(0, t.contentWidth()-t.geomInfo().view.W)
}

// clampScroll keeps both scroll offsets inside their ranges.
func (t *TextArea) clampScroll() {
	t.scrollY = minF(maxF(t.scrollY, 0), t.maxScroll(t.viewHeight()))
	t.scrollX = minF(maxF(t.scrollX, 0), t.maxScrollX())
}

// scrollCaretIntoView scrolls so the caret is visible, but only when the caret
// (or the text under it) has moved since the last time. Following it on every
// frame would snap the view back whenever the user scrolled away with the wheel
// or a scrollbar while the caret stayed put.
func (t *TextArea) scrollCaretIntoView() {
	key := caretFollow{t.caretRow, t.caretCol, t.rev, true}
	if t.followed == key {
		t.clampScroll()
		return
	}
	t.followed = key

	g := t.geomInfo()
	lh := t.lineHeight()
	rows := t.rows()
	ci := t.caretVisualIndex(rows)
	top := float64(ci) * lh
	bottom := top + lh
	if top < t.scrollY {
		t.scrollY = top
	} else if bottom > t.scrollY+g.view.H {
		t.scrollY = bottom - g.view.H
	}
	if !t.wrap {
		if f := t.face(); f != nil {
			x := t.caretXWithPreedit(f, rows[ci])
			if x < t.scrollX {
				t.scrollX = x
			} else if x+textCaretMargin > t.scrollX+g.view.W {
				t.scrollX = x + textCaretMargin - g.view.W
			}
		}
	}
	t.clampScroll()
}

// caretFollow identifies a caret state already scrolled to.
type caretFollow struct {
	row, col, rev int
	set           bool
}

// scrollTrack is the vertical scrollbar gutter along the right edge, above the
// corner when the horizontal bar is also shown.
func (t *TextArea) scrollTrack() geom.Rect {
	b := t.Bounds()
	h := b.H
	if t.geomInfo().hOver {
		h -= scrollbarWidth
	}
	return geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: h}
}

// scrollTrackH is the horizontal scrollbar gutter along the bottom edge.
func (t *TextArea) scrollTrackH() geom.Rect {
	b := t.Bounds()
	w := b.W
	if t.geomInfo().vOver {
		w -= scrollbarWidth
	}
	return geom.Rect{X: b.X, Y: b.Y + b.H - scrollbarWidth, W: w, H: scrollbarWidth}
}

func (t *TextArea) scrollMetrics() scrollMetrics {
	return scrollMetrics{content: t.contentHeight(), view: t.viewHeight(), offset: t.scrollY}
}

func (t *TextArea) scrollMetricsH() scrollMetrics {
	return scrollMetrics{content: t.contentWidth(), view: t.geomInfo().view.W, offset: t.scrollX}
}

// SetWrap turns soft word-wrap on or off at runtime (TextAreaWrap sets it at
// construction). With it off, long lines scroll sideways instead of breaking at
// the edge.
func (t *TextArea) SetWrap(on bool) {
	if t.wrap == on {
		return
	}
	t.wrap = on
	t.scrollX = 0
	t.followed = caretFollow{} // the layout changed: bring the caret back into view
	t.Invalidate()
}
