package ui

import "github.com/kpfaulkner/guie/geom"

// SetHorizontalScroll lets rows be wider than the tree, as for List: a bar appears
// along the bottom when the widest visible row (deep nodes with long labels) is
// wider than the view, a sideways wheel or Shift+wheel scrolls across, and the tree
// stops asking its parent for room for its widest row. Off by default, when long
// rows are clipped at the edge. Left and Right keep their expand/collapse meaning.
func (t *Tree) SetHorizontalScroll(on bool) {
	t.hscroll = on
	if !on {
		t.offsetX = 0
	}
	t.Invalidate()
}

// widest is the width of the widest visible row, indentation and padding
// included. Each node caches the width of its label, re-measured only when the
// label or the font changes, so asking costs a pass over the visible rows rather
// than a text measurement per row.
func (t *Tree) widest() float64 {
	f := t.face()
	if f == nil {
		return 0
	}
	fp := measureFingerprint(f)
	var w float64
	for _, r := range t.rows() {
		n := r.node
		if !n.wOK || n.wLabel != n.Label || n.wFP != fp {
			n.w, n.wLabel, n.wFP, n.wOK = f.Measure(n.Label).W, n.Label, fp, true
		}
		w = maxF(w, float64(r.depth)*treeIndent+treeChevron+treeGap+n.w)
	}
	return w + 2*treeRowPad
}

// geomInfo carves the tree's box up between rows and bars.
func (t *Tree) geomInfo() scrollGeom {
	b := t.Bounds()
	var cw float64
	if t.hscroll {
		cw = t.widest()
	}
	return carveBars(b.W, b.H, cw, t.ContentHeight(), t.hscroll)
}

func (t *Tree) maxOffsetX() float64 {
	if !t.hscroll {
		return 0
	}
	return maxF(0, t.widest()-t.geomInfo().viewW)
}

// scrollTrackH is the horizontal bar's gutter along the bottom edge.
func (t *Tree) scrollTrackH() geom.Rect {
	b := t.Bounds()
	return geom.Rect{X: b.X, Y: b.Y + b.H - scrollbarWidth, W: t.geomInfo().viewW, H: scrollbarWidth}
}

func (t *Tree) scrollMetricsH() scrollMetrics {
	return scrollMetrics{content: t.widest(), view: t.geomInfo().viewW, offset: t.offsetX}
}
