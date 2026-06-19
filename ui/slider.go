package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const (
	sliderHeight = 22
	sliderHandle = 8 // handle radius
	sliderStep   = 0.05
)

// Slider is a horizontal control for choosing a value in [0,1]. It is focusable,
// draggable with the mouse, and adjustable with Left/Right while focused.
type Slider struct {
	BaseWidget
	value    float64
	dragging bool
	hover    bool
	focused  bool
	onChange func(float64)
}

// SliderOption configures a Slider.
type SliderOption func(*Slider)

// SliderValue sets the initial value (clamped to [0,1]).
func SliderValue(v float64) SliderOption {
	return func(s *Slider) { s.value = clamp01(v) }
}

// OnSlide registers a handler called with the new value when it changes.
func OnSlide(fn func(float64)) SliderOption { return func(s *Slider) { s.onChange = fn } }

// NewSlider returns a Slider configured by opts.
func NewSlider(opts ...SliderOption) *Slider {
	s := &Slider{BaseWidget: NewBase()}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Value returns the current value in [0,1].
func (s *Slider) Value() float64 { return s.value }

// SetValue sets the value (clamped) and fires OnChange if it changed.
func (s *Slider) SetValue(v float64) {
	v = clamp01(v)
	if v == s.value {
		return
	}
	s.value = v
	if s.onChange != nil {
		s.onChange(v)
	}
}

// Focusable reports whether the slider can take focus (only when enabled).
func (s *Slider) Focusable() bool { return s.Enabled() }

// MinSize returns a default width and fixed height.
func (s *Slider) MinSize() geom.Size { return geom.Size{W: 140, H: sliderHeight} }

// track returns the horizontal extent the handle travels along.
func (s *Slider) track() (x0, x1, y float64) {
	b := s.Bounds()
	return b.X + sliderHandle, b.X + b.W - sliderHandle, b.Y + b.H/2
}

// Draw renders the track, the filled portion, and the handle.
func (s *Slider) Draw(canvas render.Canvas) {
	x0, x1, y := s.track()
	hx := x0 + (x1-x0)*s.value

	canvas.DrawLine(geom.Point{X: x0, Y: y}, geom.Point{X: x1, Y: y}, s.ColorOf(RoleBorder), 3)
	canvas.DrawLine(geom.Point{X: x0, Y: y}, geom.Point{X: hx, Y: y}, s.ColorOf(RolePrimary), 3)

	handle := s.ColorOf(RolePrimary)
	if s.hover || s.dragging {
		handle = s.ColorOf(RoleAccent)
	}
	canvas.FillCircle(geom.Point{X: hx, Y: y}, sliderHandle, handle)
	if s.focused {
		canvas.StrokeCircle(geom.Point{X: hx, Y: y}, sliderHandle+2, s.ColorOf(RoleAccent), 1)
	}
}

// setFromX maps an absolute x coordinate to a value in [0,1].
func (s *Slider) setFromX(x float64) {
	x0, x1, _ := s.track()
	if x1 <= x0 {
		return
	}
	s.SetValue((x - x0) / (x1 - x0))
}

// HandleEvent drives dragging and keyboard adjustment.
func (s *Slider) HandleEvent(ev *Event) bool {
	if !s.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		s.hover = true
		return true
	case EventPointerLeave:
		s.hover = false
		return true
	case EventPointerDown:
		s.dragging = true
		s.setFromX(ev.Pos.X)
		return true
	case EventPointerMove:
		if s.dragging {
			s.setFromX(ev.Pos.X)
			return true
		}
	case EventPointerUp:
		s.dragging = false
		return true
	case EventFocusGained:
		s.focused = true
		return true
	case EventFocusLost:
		s.focused = false
		s.dragging = false
		return true
	case EventKeyDown:
		switch ev.Key {
		case render.KeyLeft:
			s.SetValue(s.value - sliderStep)
			return true
		case render.KeyRight:
			s.SetValue(s.value + sliderStep)
			return true
		}
	}
	return false
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
