package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const (
	scrollbarWidth = 12
	wheelStep      = 32 // logical pixels scrolled per wheel notch
	minThumb       = 24
)

// ScrollView shows a single content widget that may be taller than the view,
// clipping it to a viewport and offsetting it vertically. The content scrolls
// with the mouse wheel (events bubble up from the content) and by dragging the
// scrollbar thumb on the right edge. Horizontal scrolling is not yet supported.
type ScrollView struct {
	BaseWidget
	content Widget
	offsetY float64

	dragging   bool
	dragStartY float64
	dragStartO float64
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

// Children returns the content widget (for hit-testing and event bubbling).
func (s *ScrollView) Children() []Widget {
	if s.content == nil {
		return nil
	}
	return []Widget{s.content}
}

func (s *ScrollView) mount(self, parent Widget, ctx *treeContext) {
	s.BaseWidget.mount(self, parent, ctx)
	if s.content != nil {
		s.content.mount(s.content, self, ctx)
	}
}

// viewport is the content area, excluding the scrollbar gutter.
func (s *ScrollView) viewport() geom.Rect {
	b := s.Bounds()
	return geom.Rect{X: b.X, Y: b.Y, W: b.W - scrollbarWidth, H: b.H}
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

func (s *ScrollView) maxOffset() float64 {
	return maxF(0, s.contentHeight()-s.viewport().H)
}

func (s *ScrollView) clamp() {
	if s.offsetY < 0 {
		s.offsetY = 0
	}
	if m := s.maxOffset(); s.offsetY > m {
		s.offsetY = m
	}
}

// minViewport is the default intrinsic height of a ScrollView. A scroll view is
// meant to be smaller than its content, so MinSize must not report the content
// height (that would let it grow to the full content size and leave nothing to
// scroll). The parent is expected to give it space via stretch/weight.
const minViewport = 48

// MinSize requests the content width plus the scrollbar gutter, and only a
// modest intrinsic height — the view scrolls to reveal taller content.
func (s *ScrollView) MinSize() geom.Size {
	if s.content == nil {
		return geom.Size{W: scrollbarWidth, H: minViewport}
	}
	return geom.Size{W: s.content.MinSize().W + scrollbarWidth, H: minViewport}
}

// Layout positions the content within the viewport, offset by the scroll
// amount, and clamps the offset to the current content height.
func (s *ScrollView) Layout() {
	if s.content == nil {
		return
	}
	s.clamp()
	vp := s.viewport()
	s.content.SetBounds(geom.Rect{X: vp.X, Y: vp.Y - s.offsetY, W: vp.W, H: s.contentHeight()})
	s.content.Layout()
}

// thumbRect computes the scrollbar thumb rectangle for the current offset.
func (s *ScrollView) thumbRect() geom.Rect {
	b := s.Bounds()
	vp := s.viewport()
	ch := s.contentHeight()
	thumbH := maxF(minThumb, vp.H*vp.H/ch)
	var t float64
	if m := s.maxOffset(); m > 0 {
		t = (s.offsetY / m) * (vp.H - thumbH)
	}
	return geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y + t, W: scrollbarWidth, H: thumbH}
}

// Draw paints the content (clipped to the viewport) and the scrollbar.
func (s *ScrollView) Draw(canvas render.Canvas) {
	b := s.Bounds()
	canvas.FillRect(b, s.ColorOf(RoleSurface))

	if s.content != nil {
		vp := s.viewport()
		canvas.PushClip(vp)
		s.content.Draw(canvas)
		canvas.PopClip()
	}

	// Scrollbar gutter and thumb.
	gutter := geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: b.H}
	canvas.FillRect(gutter, s.ColorOf(RoleBackground))
	canvas.FillRect(s.thumbRect(), s.ColorOf(RoleAccent))
}

// HandleEvent scrolls on the wheel and drags the thumb.
func (s *ScrollView) HandleEvent(ev *Event) bool {
	switch ev.Type {
	case EventWheel:
		s.offsetY -= ev.Wheel.Y * wheelStep
		s.clamp()
		s.Invalidate()
		return true
	case EventPointerDown:
		if s.thumbRect().Contains(ev.Pos) {
			s.dragging = true
			s.dragStartY = ev.Pos.Y
			s.dragStartO = s.offsetY
			return true
		}
		// Consume presses on the gutter so they don't fall through.
		return ev.Pos.X >= s.Bounds().X+s.Bounds().W-scrollbarWidth
	case EventPointerMove:
		if s.dragging {
			s.dragByThumb(ev.Pos.Y - s.dragStartY)
			return true
		}
	case EventPointerUp:
		if s.dragging {
			s.dragging = false
			return true
		}
	}
	return false
}

// dragByThumb converts a thumb drag delta (in pixels) into a content offset.
func (s *ScrollView) dragByThumb(dy float64) {
	vp := s.viewport()
	thumbH := s.thumbRect().H
	travel := vp.H - thumbH
	if travel <= 0 {
		return
	}
	s.offsetY = s.dragStartO + dy*(s.maxOffset()/travel)
	s.clamp()
	s.Invalidate()
}
