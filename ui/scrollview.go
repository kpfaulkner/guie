package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ScrollSpeed is the number of pixels scrolled per wheel notch.
const ScrollSpeed = 24

// ScrollView is a viewport that clips a single (typically larger) content
// widget and lets the user scroll it with the mouse wheel. The content's
// Bounds define its full size; the portion visible is determined by the
// current scroll offset.
type ScrollView struct {
	BaseWidget

	// Background, if non-nil, fills the viewport before the content is drawn.
	Background color.Color

	content Widget
	offset  Point
}

// NewScrollView returns an empty scroll view occupying r.
func NewScrollView(r Rect) *ScrollView {
	return &ScrollView{
		BaseWidget: NewBase(r),
		Background: DefaultWindowBackground,
	}
}

// SetContent sets the widget displayed inside the viewport.
func (s *ScrollView) SetContent(w Widget) { s.content = w }

// Offset returns the current scroll offset.
func (s *ScrollView) Offset() Point { return s.offset }

// Update advances the content widget.
func (s *ScrollView) Update() error {
	if s.content != nil {
		return s.content.Update()
	}
	return nil
}

// maxOffset returns the largest valid scroll offset given the content size.
func (s *ScrollView) maxOffset() Point {
	if s.content == nil {
		return Point{}
	}
	cb := s.content.Bounds()
	return Point{
		X: max(0, cb.W-s.bounds.W),
		Y: max(0, cb.H-s.bounds.H),
	}
}

// Draw fills the background and draws the clipped content shifted by the
// scroll offset.
func (s *ScrollView) Draw(dst *ebiten.Image, origin Point) {
	abs := s.bounds.Add(origin)
	if s.Background != nil {
		vector.FillRect(dst, float32(abs.X), float32(abs.Y), float32(abs.W), float32(abs.H), s.Background, false)
	}
	if s.content == nil {
		return
	}

	// Clip drawing to the viewport using a sub-image; sub-images share the
	// parent's coordinate system, so content coordinates need no adjustment.
	clip := image.Rect(abs.X, abs.Y, abs.X+abs.W, abs.Y+abs.H)
	view := dst.SubImage(clip).(*ebiten.Image)

	contentOrigin := Point{X: abs.X - s.offset.X, Y: abs.Y - s.offset.Y}
	s.content.Draw(view, contentOrigin)
}

// HandleEvent consumes wheel events to scroll, and forwards other events to the
// content in its scrolled coordinate space.
func (s *ScrollView) HandleEvent(ev *Event, origin Point) bool {
	abs := s.bounds.Add(origin)
	inside := abs.Contains(ev.Pos)

	if ev.Type == MouseWheel && inside {
		mx := s.maxOffset()
		s.offset.Y = clamp(s.offset.Y-int(ev.WheelY*ScrollSpeed), 0, mx.Y)
		s.offset.X = clamp(s.offset.X-int(ev.WheelX*ScrollSpeed), 0, mx.X)
		return true
	}

	if s.content == nil {
		return false
	}
	// Only forward positional mouse events that fall within the viewport.
	if (ev.Type == MouseMove || ev.Type == MouseDown || ev.Type == MouseUp) && !inside {
		return false
	}
	contentOrigin := Point{X: abs.X - s.offset.X, Y: abs.Y - s.offset.Y}
	return s.content.HandleEvent(ev, contentOrigin)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
