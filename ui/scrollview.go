package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	scrollbarWidth = 12
	wheelStep      = 32 // logical pixels scrolled per wheel notch
	minThumb       = 24
)

// ScrollView shows a single content widget that may be larger than the view,
// clipping it to a viewport and offsetting it. The content scrolls vertically
// with the mouse wheel (events bubble up from the content) and by dragging or
// clicking the scrollbar on the right edge (thumb drag, gutter paging).
//
// Horizontal scrolling is off by default, so the content is stretched to the
// view's width as always. SetHorizontalScroll turns it on: the content then keeps
// its natural width, and a second bar appears along the bottom whenever it is
// wider than the view. Shift+wheel, or a sideways wheel, scrolls it.
type ScrollView struct {
	BaseWidget
	content Widget
	offsetX float64
	offsetY float64
	hscroll bool

	barV barDrag // vertical scrollbar thumb-drag state
	barH barDrag // horizontal scrollbar thumb-drag state
}

// NewScrollView returns an empty ScrollView.
func NewScrollView() *ScrollView { return &ScrollView{BaseWidget: NewBase()} }

// SetContent sets the scrollable content widget.
func (s *ScrollView) SetContent(w Widget) {
	s.content = w
	if s.ctx != nil && w != nil {
		w.mount(w, s.self, s.ctx)
	}
	s.Invalidate()
}

// SetHorizontalScroll turns horizontal scrolling on or off. When on, the content
// keeps its minimum width instead of being stretched to the view, and the view no
// longer asks its parent for room for all of it (like its height, the width is the
// parent's to give). Turning it off scrolls back to the left edge.
func (s *ScrollView) SetHorizontalScroll(on bool) {
	s.hscroll = on
	if !on {
		s.offsetX = 0
	}
	s.Invalidate()
}

// Children returns the content widget (for hit-testing and event bubbling).
func (s *ScrollView) Children() []Widget {
	if s.content == nil {
		return nil
	}
	return []Widget{s.content}
}

// childClip restricts hit-testing of the content to the viewport. Scrolled
// content can extend under the scrollbars, which must keep their own events.
func (s *ScrollView) childClip() geom.Rect { return s.viewport() }

func (s *ScrollView) mount(self, parent Widget, ctx *treeContext) {
	s.BaseWidget.mount(self, parent, ctx)
	if s.content != nil {
		s.content.mount(s.content, self, ctx)
	}
}

// hbarShown reports whether the horizontal bar is present: horizontal scrolling
// is on and the content is wider than the area left of the vertical gutter. It
// depends only on widths, so it cannot disagree with the viewport height it
// shrinks.
func (s *ScrollView) hbarShown() bool {
	if !s.hscroll || s.content == nil {
		return false
	}
	return s.content.MinSize().W > s.Bounds().W-scrollbarWidth
}

// viewport is the content area, excluding the scrollbar gutters.
func (s *ScrollView) viewport() geom.Rect {
	b := s.Bounds()
	vp := geom.Rect{X: b.X, Y: b.Y, W: b.W - scrollbarWidth, H: b.H}
	if s.hbarShown() {
		vp.H -= scrollbarWidth
	}
	return vp
}

// contentHeight is the laid-out height of the content, at least the viewport
// height.
func (s *ScrollView) contentHeight() float64 {
	vp := s.viewport()
	if s.content == nil {
		return vp.H
	}
	return maxF(s.content.MinSize().H, vp.H)
}

// contentWidth is the laid-out width of the content: the viewport width unless
// horizontal scrolling lets it keep its natural width.
func (s *ScrollView) contentWidth() float64 {
	vp := s.viewport()
	if s.content == nil || !s.hscroll {
		return vp.W
	}
	return maxF(s.content.MinSize().W, vp.W)
}

func (s *ScrollView) maxOffset() float64 {
	return maxF(0, s.contentHeight()-s.viewport().H)
}

func (s *ScrollView) maxOffsetX() float64 {
	return maxF(0, s.contentWidth()-s.viewport().W)
}

func (s *ScrollView) clamp() {
	s.offsetY = minF(maxF(s.offsetY, 0), s.maxOffset())
	s.offsetX = minF(maxF(s.offsetX, 0), s.maxOffsetX())
}

// minViewport is the default intrinsic height of a ScrollView. A scroll view is
// meant to be smaller than its content, so MinSize must not report the content
// height (that would let it grow to the full content size and leave nothing to
// scroll). The parent is expected to give it space via stretch/weight.
const minViewport = 48

// MinSize requests the content width plus the scrollbar gutter, and only a
// modest intrinsic height — the view scrolls to reveal taller content. With
// horizontal scrolling on, the width is modest too, for the same reason.
func (s *ScrollView) MinSize() geom.Size {
	if s.content == nil {
		return geom.Size{W: scrollbarWidth, H: minViewport}
	}
	if s.hscroll {
		return geom.Size{W: minViewport + scrollbarWidth, H: minViewport}
	}
	return geom.Size{W: s.content.MinSize().W + scrollbarWidth, H: minViewport}
}

// Layout positions the content within the viewport, offset by the scroll
// amount, and clamps the offsets to the current content size.
func (s *ScrollView) Layout() {
	if s.content == nil {
		return
	}
	s.clamp()
	vp := s.viewport()
	s.content.SetBounds(geom.Rect{
		X: vp.X - s.offsetX, Y: vp.Y - s.offsetY,
		W: s.contentWidth(), H: s.contentHeight(),
	})
	s.content.Layout()
}

// track is the vertical scrollbar gutter along the right edge, above the corner
// when the horizontal bar is also shown.
func (s *ScrollView) track() geom.Rect {
	b := s.Bounds()
	return geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: s.viewport().H}
}

// trackH is the horizontal scrollbar gutter along the bottom edge (only meaningful
// while hbarShown).
func (s *ScrollView) trackH() geom.Rect {
	b := s.Bounds()
	return geom.Rect{X: b.X, Y: b.Y + b.H - scrollbarWidth, W: s.viewport().W, H: scrollbarWidth}
}

func (s *ScrollView) metrics() scrollMetrics {
	return scrollMetrics{content: s.contentHeight(), view: s.viewport().H, offset: s.offsetY}
}

func (s *ScrollView) metricsH() scrollMetrics {
	return scrollMetrics{content: s.contentWidth(), view: s.viewport().W, offset: s.offsetX}
}

// Draw paints the content (clipped to the viewport) and the scrollbars.
func (s *ScrollView) Draw(canvas render.Canvas) {
	b := s.Bounds()
	canvas.FillRect(b, s.ColourOf(RoleSurface))

	if s.content != nil {
		canvas.PushClip(s.viewport())
		s.content.Draw(canvas)
		canvas.PopClip()
	}

	gutter, thumb := s.ColourOf(RoleBackground), s.ColourOf(RoleAccent)
	drawBar(canvas, s.track(), true, s.metrics(), gutter, thumb)
	if s.hbarShown() {
		drawBar(canvas, s.trackH(), false, s.metricsH(), gutter, thumb)
		// The corner where the two gutters meet.
		canvas.FillRect(geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y + b.H - scrollbarWidth, W: scrollbarWidth, H: scrollbarWidth}, gutter)
	}
}

// HandleEvent scrolls on the wheel and drags or pages with the scrollbars.
func (s *ScrollView) HandleEvent(ev *Event) bool {
	if ev.Type == EventWheel {
		if s.hscroll {
			// A sideways wheel, or Shift with the ordinary one, scrolls across.
			dx := ev.Wheel.X
			if dx == 0 && ev.Modifiers.Has(render.ModShift) {
				dx = ev.Wheel.Y
			}
			if dx != 0 && s.maxOffsetX() > 0 {
				s.offsetX -= dx * wheelStep
				s.clamp()
				s.Invalidate()
				return true
			}
		}
		s.offsetY -= ev.Wheel.Y * wheelStep
		s.clamp()
		s.Invalidate()
		return true
	}
	if off, changed, ok := s.barV.handle(ev, s.track(), true, true, s.metrics()); ok {
		if changed {
			s.offsetY = off
			s.clamp()
			s.Invalidate()
		}
		return true
	}
	if off, changed, ok := s.barH.handle(ev, s.trackH(), s.hbarShown(), false, s.metricsH()); ok {
		if changed {
			s.offsetX = off
			s.clamp()
			s.Invalidate()
		}
		return true
	}
	return false
}
