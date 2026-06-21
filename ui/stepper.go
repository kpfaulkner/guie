package ui

import (
	"image/color"
	"strconv"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// stepper layout constants.
var stepperPadding = geom.Insets{Top: 5, Right: 6, Bottom: 5, Left: 8}

const stepperButtonW = 18 // width of the up/down button column

// Stepper is a numeric input adjusted with its up/down buttons, the Up/Down
// arrow keys while focused, or the mouse wheel. The value is held in [Min, Max]
// and changed by Step; OnChange fires whenever it changes. It is a self-drawn
// widget (no free-form text entry): values stay valid by construction, so there
// is nothing to parse or reject.
type Stepper struct {
	BaseWidget
	value    float64
	min      float64
	max      float64
	step     float64
	decimals int

	focused   bool
	hover     bool
	hoverUp   bool
	hoverDown bool

	font     render.FontFace
	onChange func(float64)
}

// StepperOption configures a Stepper.
type StepperOption func(*Stepper)

// StepperRange sets the inclusive value bounds.
func StepperRange(min, max float64) StepperOption {
	return func(s *Stepper) { s.min, s.max = min, max }
}

// StepperStep sets the increment applied by each step.
func StepperStep(step float64) StepperOption {
	return func(s *Stepper) {
		if step > 0 {
			s.step = step
		}
	}
}

// StepperValue sets the initial value (clamped to the range).
func StepperValue(v float64) StepperOption {
	return func(s *Stepper) { s.value = v }
}

// StepperDecimals sets how many fractional digits are displayed.
func StepperDecimals(n int) StepperOption {
	return func(s *Stepper) {
		if n >= 0 {
			s.decimals = n
		}
	}
}

// NewStepper returns a Stepper. By default it ranges over [0, 100] with a step
// of 1 and no decimals.
func NewStepper(opts ...StepperOption) *Stepper {
	s := &Stepper{BaseWidget: NewBase(), min: 0, max: 100, step: 1}
	for _, o := range opts {
		o(s)
	}
	s.value = clampF(s.value, s.min, s.max)
	return s
}

// OnChange registers the handler invoked whenever the value changes.
func (s *Stepper) OnChange(fn func(float64)) { s.onChange = fn }

// Value returns the current value.
func (s *Stepper) Value() float64 { return s.value }

// SetValue sets the value (clamped to the range) and fires OnChange if it
// changed.
func (s *Stepper) SetValue(v float64) {
	v = clampF(v, s.min, s.max)
	if v == s.value {
		return
	}
	s.value = v
	if s.onChange != nil {
		s.onChange(v)
	}
	s.Invalidate()
}

// Step adjusts the value by n increments (negative to decrease).
func (s *Stepper) Step(n int) { s.SetValue(s.value + float64(n)*s.step) }

func (s *Stepper) face() render.FontFace {
	if s.font != nil {
		return s.font
	}
	return s.appTheme().Font
}

// SetFont overrides the stepper's font face (nil falls back to the theme font).
func (s *Stepper) SetFont(f render.FontFace) {
	s.font = f
	s.Invalidate()
}

// Focusable reports whether the stepper can take focus (only when enabled).
func (s *Stepper) Focusable() bool { return s.Enabled() }

func (s *Stepper) text() string {
	return strconv.FormatFloat(s.value, 'f', s.decimals, 64)
}

// MinSize returns a width sized to the widest value in the range plus the button
// column, and one line tall.
func (s *Stepper) MinSize() geom.Size {
	f := s.face()
	var h, w float64
	if f != nil {
		h = f.Measure("Ag").H
		lo := strconv.FormatFloat(s.min, 'f', s.decimals, 64)
		hi := strconv.FormatFloat(s.max, 'f', s.decimals, 64)
		w = maxF(f.Measure(lo).W, f.Measure(hi).W)
	}
	return geom.Size{
		W: w + stepperPadding.Left + stepperPadding.Right + stepperButtonW,
		H: h + stepperPadding.Top + stepperPadding.Bottom,
	}
}

// buttonColumn returns the rectangle of the up/down button column.
func (s *Stepper) buttonColumn() geom.Rect {
	b := s.Bounds()
	return geom.Rect{X: b.X + b.W - stepperButtonW, Y: b.Y, W: stepperButtonW, H: b.H}
}

// Draw paints the box, the value, and the up/down buttons with hover/disabled
// states.
func (s *Stepper) Draw(canvas render.Canvas) {
	f := s.face()
	if f == nil {
		return
	}
	b := s.Bounds()
	rad := s.cornerRadius()

	canvas.FillRoundRect(b, rad, s.ColorOf(RoleSurface))
	border := s.ColorOf(RoleBorder)
	if s.focused {
		border = s.ColorOf(RoleAccent)
	}
	canvas.StrokeRoundRect(b, rad, border, 1)

	textColor := s.ColorOf(RoleText)
	if !s.Enabled() {
		textColor = s.ColorOf(RoleDisabled)
	}
	inner := b.Inset(stepperPadding)
	canvas.DrawText(s.text(), geom.Point{X: inner.X, Y: vCenterY(f, inner.Y, inner.H)}, f, textColor)

	// Button column: up on the top half, down on the bottom half.
	col := s.buttonColumn()
	canvas.DrawLine(geom.Point{X: col.X, Y: b.Y + 2}, geom.Point{X: col.X, Y: b.Y + b.H - 2}, s.ColorOf(RoleBorder), 1)
	midY := col.Y + col.H/2
	upRect := geom.Rect{X: col.X, Y: col.Y, W: col.W, H: col.H / 2}
	downRect := geom.Rect{X: col.X, Y: midY, W: col.W, H: col.H / 2}
	if s.hoverUp {
		canvas.FillRect(upRect, lighten(s.ColorOf(RoleSurface), 1.25))
	}
	if s.hoverDown {
		canvas.FillRect(downRect, lighten(s.ColorOf(RoleSurface), 1.25))
	}
	arrow := s.ColorOf(RoleText)
	if !s.Enabled() {
		arrow = s.ColorOf(RoleDisabled)
	}
	s.drawArrow(canvas, upRect, true, arrow)
	s.drawArrow(canvas, downRect, false, arrow)
}

// drawArrow draws a small up- or down-pointing chevron centered in r.
func (s *Stepper) drawArrow(canvas render.Canvas, r geom.Rect, up bool, col color.Color) {
	cx := r.X + r.W/2
	cy := r.Y + r.H/2
	const w = 3.0
	if up {
		canvas.DrawLine(geom.Point{X: cx - w, Y: cy + w/2}, geom.Point{X: cx, Y: cy - w/2}, col, 1.5)
		canvas.DrawLine(geom.Point{X: cx + w, Y: cy + w/2}, geom.Point{X: cx, Y: cy - w/2}, col, 1.5)
	} else {
		canvas.DrawLine(geom.Point{X: cx - w, Y: cy - w/2}, geom.Point{X: cx, Y: cy + w/2}, col, 1.5)
		canvas.DrawLine(geom.Point{X: cx + w, Y: cy - w/2}, geom.Point{X: cx, Y: cy + w/2}, col, 1.5)
	}
}

// HandleEvent steps the value in response to button clicks, the arrow keys, and
// the wheel, and tracks hover state for the buttons.
func (s *Stepper) HandleEvent(ev *Event) bool {
	if !s.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		s.hover = true
		return true
	case EventPointerLeave:
		s.hover = false
		s.hoverUp, s.hoverDown = false, false
		return true
	case EventPointerMove:
		col := s.buttonColumn()
		in := col.Contains(ev.Pos)
		s.hoverUp = in && ev.Pos.Y < col.Y+col.H/2
		s.hoverDown = in && ev.Pos.Y >= col.Y+col.H/2
		return true
	case EventClick:
		col := s.buttonColumn()
		if col.Contains(ev.Pos) {
			if ev.Pos.Y < col.Y+col.H/2 {
				s.Step(1)
			} else {
				s.Step(-1)
			}
		}
		return true
	case EventWheel:
		if ev.Wheel.Y > 0 {
			s.Step(1)
		} else if ev.Wheel.Y < 0 {
			s.Step(-1)
		}
		return true
	case EventFocusGained:
		s.focused = true
		return true
	case EventFocusLost:
		s.focused = false
		return true
	case EventKeyDown:
		switch ev.Key {
		case render.KeyUp:
			s.Step(1)
		case render.KeyDown:
			s.Step(-1)
		case render.KeyHome:
			s.SetValue(s.min)
		case render.KeyEnd:
			s.SetValue(s.max)
		default:
			return false
		}
		return true
	}
	return false
}

func clampF(v, lo, hi float64) float64 {
	if lo <= hi {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
	}
	return v
}
