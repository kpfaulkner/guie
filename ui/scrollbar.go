package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// The scrollbar is built in two layers. The core (scrollMetrics, thumbRect,
// drawBar, barDrag) is plain geometry and drag state with no widget in it, so the
// self-drawing widgets (List, Tree, Table, TextArea, ScrollView), which have no
// child widgets to hang a bar on, all share one implementation of how a bar looks
// and behaves. ScrollBar is the same core as a widget in its own right, for custom
// surfaces such as a CanvasWidget.

// scrollMetrics describes what a bar scrolls: the total content length, the
// visible length, and the current offset of the visible window, all along the bar's
// axis in logical pixels.
type scrollMetrics struct {
	content, view, offset float64
}

func (m scrollMetrics) maxOffset() float64 { return maxF(0, m.content-m.view) }

func (m scrollMetrics) clamped(off float64) float64 {
	return minF(maxF(off, 0), m.maxOffset())
}

// axisLen and axisPos pick the bar's length and a point's coordinate along it.
func axisLen(r geom.Rect, vertical bool) float64 {
	if vertical {
		return r.H
	}
	return r.W
}

func axisPos(p geom.Point, vertical bool) float64 {
	if vertical {
		return p.Y
	}
	return p.X
}

// thumbLen is the thumb's length along a track of the given length: the share of
// the track the visible part is of the content, never shorter than minThumb (or
// the track itself, if that is shorter) and the whole track when nothing scrolls.
func thumbLen(length float64, m scrollMetrics) float64 {
	if m.content <= 0 || m.view >= m.content {
		return length
	}
	return minF(length, maxF(minF(minThumb, length), length*m.view/m.content))
}

// thumbRect returns the thumb inside track for the given metrics.
func thumbRect(track geom.Rect, vertical bool, m scrollMetrics) geom.Rect {
	length := axisLen(track, vertical)
	tl := thumbLen(length, m)
	var pos float64
	if mx := m.maxOffset(); mx > 0 {
		pos = m.clamped(m.offset) / mx * (length - tl)
	}
	if vertical {
		return geom.Rect{X: track.X, Y: track.Y + pos, W: track.W, H: tl}
	}
	return geom.Rect{X: track.X + pos, Y: track.Y, W: tl, H: track.H}
}

// drawBar paints the gutter and the thumb.
func drawBar(c render.Canvas, track geom.Rect, vertical bool, m scrollMetrics, gutter, thumb color.Color) {
	c.FillRect(track, gutter)
	c.FillRect(thumbRect(track, vertical, m), thumb)
}

// barDrag is the pointer state of one bar: whether the thumb is being dragged,
// and where the drag started.
type barDrag struct {
	active   bool
	startPos float64 // pointer coordinate along the axis at the press
	startOff float64 // content offset at the press
}

// handle runs a bar's pointer behaviour for ev. track is the bar's rectangle and
// show whether it is currently there (a list that fits has no bar). It reports the
// new offset, whether that differs from m.offset, and whether the event was the
// bar's, in which case the owner must not treat it as its own:
//
//   - a left press on the thumb starts a drag; moves while dragging scroll the
//     content in proportion, even when the pointer leaves the bar; release ends it;
//   - a left press on the gutter beside the thumb pages one view towards it;
//   - the click that follows any press on the bar is swallowed, so a row or a
//     caret under the bar is not selected through it.
func (d *barDrag) handle(ev *Event, track geom.Rect, show, vertical bool, m scrollMetrics) (off float64, changed, consumed bool) {
	switch ev.Type {
	case EventPointerDown:
		if !show || ev.Button != render.MouseLeft || !track.Contains(ev.Pos) {
			return m.offset, false, false
		}
		th := thumbRect(track, vertical, m)
		if th.Contains(ev.Pos) {
			d.active, d.startPos, d.startOff = true, axisPos(ev.Pos, vertical), m.offset
			return m.offset, false, true
		}
		off = m.offset + m.view
		if axisPos(ev.Pos, vertical) < axisPos(th.Min(), vertical) {
			off = m.offset - m.view
		}
		off = m.clamped(off)
		return off, off != m.offset, true
	case EventPointerMove:
		if !d.active {
			return m.offset, false, false
		}
		travel := axisLen(track, vertical) - thumbLen(axisLen(track, vertical), m)
		if travel <= 0 {
			return m.offset, false, true
		}
		off = m.clamped(d.startOff + (axisPos(ev.Pos, vertical)-d.startPos)*(m.maxOffset()/travel))
		return off, off != m.offset, true
	case EventPointerUp:
		if d.active {
			d.active = false
			return m.offset, false, true
		}
	case EventClick:
		if show && track.Contains(ev.Pos) {
			return m.offset, false, true
		}
	}
	return m.offset, false, false
}

// ScrollBar is a standalone scroll bar. It does not scroll anything itself: the
// owner tells it how much content there is and how much is visible with SetRange,
// the bar reports the offset the user scrolls to through OnScroll, and the owner
// applies it. Use it to scroll a CanvasWidget or any custom surface; the built-in
// scrolling widgets draw the same bar themselves.
//
// The user scrolls by dragging the thumb, clicking the gutter beside it (one view
// per click) or using the wheel over the bar. The bar fills its Bounds, so size it
// like any widget: it asks for a scrollbarWidth-thick strip along the cross axis.
type ScrollBar struct {
	BaseWidget
	vertical bool
	m        scrollMetrics
	drag     barDrag
	onScroll func(offset float64)
}

// NewScrollBar returns a scroll bar along dir: geom.Vertical for a bar on the
// right of a surface, geom.Horizontal for one along the bottom.
func NewScrollBar(dir geom.Direction) *ScrollBar {
	return &ScrollBar{BaseWidget: NewBase(), vertical: dir == geom.Vertical}
}

// SetRange sets the total content length and the visible length, in the same units
// as the offset (normally logical pixels). The offset is clamped to the new range.
func (s *ScrollBar) SetRange(content, view float64) {
	s.m.content, s.m.view = maxF(content, 0), maxF(view, 0)
	s.m.offset = s.m.clamped(s.m.offset)
}

// Offset returns the current scroll offset: how far the visible window is from the
// start of the content.
func (s *ScrollBar) Offset() float64 { return s.m.offset }

// SetOffset moves the thumb to offset (clamped to the range). It does not call
// OnScroll: the owner is the one asking.
func (s *ScrollBar) SetOffset(offset float64) { s.m.offset = s.m.clamped(offset) }

// OnScroll registers the handler called with the new offset when the user scrolls
// the bar.
func (s *ScrollBar) OnScroll(fn func(offset float64)) { s.onScroll = fn }

// MinSize is a scrollbarWidth-thick strip, long enough for a minimum thumb.
func (s *ScrollBar) MinSize() geom.Size {
	if s.vertical {
		return geom.Size{W: scrollbarWidth, H: 2 * minThumb}
	}
	return geom.Size{W: 2 * minThumb, H: scrollbarWidth}
}

// Draw paints the gutter and thumb.
func (s *ScrollBar) Draw(c render.Canvas) {
	drawBar(c, s.Bounds(), s.vertical, s.m, s.ColourOf(RoleBackground), s.ColourOf(RoleAccent))
}

// scrollTo applies a new offset and tells the owner.
func (s *ScrollBar) scrollTo(off float64) {
	off = s.m.clamped(off)
	if off == s.m.offset {
		return
	}
	s.m.offset = off
	if s.onScroll != nil {
		s.onScroll(off)
	}
}

// HandleEvent handles thumb drags, gutter paging and the wheel.
func (s *ScrollBar) HandleEvent(ev *Event) bool {
	if !s.Enabled() {
		return false
	}
	if off, changed, ok := s.drag.handle(ev, s.Bounds(), true, s.vertical, s.m); ok {
		if changed {
			s.scrollTo(off)
		}
		return true
	}
	if ev.Type == EventWheel {
		d := ev.Wheel.Y
		if !s.vertical && ev.Wheel.X != 0 {
			d = ev.Wheel.X
		}
		s.scrollTo(s.m.offset - d*wheelStep)
		return true
	}
	return false
}

// wheelAcross is the wheel movement that should scroll sideways: a horizontal
// wheel, or the ordinary one while Shift is held. Zero means none.
func wheelAcross(ev *Event) float64 {
	if ev.Wheel.X != 0 {
		return ev.Wheel.X
	}
	if ev.Modifiers.Has(render.ModShift) {
		return ev.Wheel.Y
	}
	return 0
}

// scrollGeom is how a w x h box with content of a given size is carved up between
// the visible area and the scrollbars. The bars depend on each other: the vertical
// one narrows the view, which can make the content overflow sideways and need the
// horizontal one, whose strip shortens the view and can make the rows overflow. A
// bar only ever takes space, so one extra pass settles it.
type scrollGeom struct {
	vOver, hOver bool
	viewW, viewH float64
}

// carveBars lays out the bars for a box of w x h. The horizontal bar is only ever
// shown when hscroll is on.
func carveBars(w, h, contentW, contentH float64, hscroll bool) scrollGeom {
	g := scrollGeom{viewW: w, viewH: h}
	g.vOver = contentH > h
	if g.vOver {
		g.viewW -= scrollbarWidth
	}
	if hscroll && contentW > g.viewW {
		g.hOver = true
		g.viewH = h - scrollbarWidth
		g.vOver = contentH > g.viewH
		g.viewW = w
		if g.vOver {
			g.viewW -= scrollbarWidth
		}
	}
	return g
}

// measureFingerprint is a cheap number that changes when the font does, so a cache
// of text widths can tell it is stale without comparing font values.
func measureFingerprint(f render.FontFace) float64 {
	s := f.Measure("Mgq0 ")
	return s.W*1000 + s.H
}
