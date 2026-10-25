package ui

import "github.com/kpfaulkner/guie/geom"

// SetHorizontalScroll lets rows be wider than the list. The rows then keep their
// natural width, a bar appears along the bottom when the widest is wider than the
// view, and Left/Right (or a sideways wheel, or Shift+wheel) scroll across. The
// list also stops asking its parent for room for its widest row. Off by default,
// when long rows are clipped at the edge.
func (l *List) SetHorizontalScroll(on bool) {
	l.hscroll = on
	if !on {
		l.offsetX = 0
	}
	l.Invalidate()
}

// widest is the width of the widest row plus its padding. Widths are cached per
// item and re-measured only for items whose text changed (the caller owns the
// slice and may edit it) or when the font did, so asking is cheap even for long
// lists.
func (l *List) widest() float64 {
	f := l.face()
	if f == nil {
		return 0
	}
	if fp := measureFingerprint(f); fp != l.wFP || len(l.wStr) != len(l.items) {
		l.wFP = fp
		l.wStr = make([]string, len(l.items))
		l.wW = make([]float64, len(l.items))
		for i := range l.wStr {
			l.wStr[i] = "\x00" // never equal to a real item, so it is measured below
		}
	}
	var w float64
	for i, it := range l.items {
		if l.wStr[i] != it {
			l.wStr[i], l.wW[i] = it, f.Measure(it).W
		}
		w = maxF(w, l.wW[i])
	}
	return w + 2*l.px(listRowPad)
}

// geomInfo carves the list's box up between rows and bars.
func (l *List) geomInfo() scrollGeom {
	b := l.Bounds()
	var cw float64
	if l.hscroll {
		cw = l.widest()
	}
	return carveBars(b.W, b.H, cw, l.ContentHeight(), l.hscroll)
}

func (l *List) maxOffsetX() float64 {
	if !l.hscroll {
		return 0
	}
	return maxF(0, l.widest()-l.geomInfo().viewW)
}

// scrollTrackH is the horizontal bar's gutter along the bottom edge.
func (l *List) scrollTrackH() geom.Rect {
	b := l.Bounds()
	return geom.Rect{X: b.X, Y: b.Y + b.H - scrollbarWidth, W: l.geomInfo().viewW, H: scrollbarWidth}
}

func (l *List) scrollMetricsH() scrollMetrics {
	return scrollMetrics{content: l.widest(), view: l.geomInfo().viewW, offset: l.offsetX}
}
