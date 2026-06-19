package ui

import (
	"image/color"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// buttonPadding is the space reserved around a button's label, used by MinSize.
var buttonPadding = geom.Insets{Top: 6, Right: 14, Bottom: 6, Left: 14}

// Button is a clickable widget that draws a themed background and a centered
// label, reacting to hover and press. Click handling is delivered through the
// pointer events dispatched by the App: a click fires when the pointer is
// released over the button after being pressed on it.
type Button struct {
	BaseWidget
	text    string
	onClick func()

	hover   bool
	pressed bool
	focused bool

	font render.FontFace
}

// ButtonOption configures a Button during NewButton.
type ButtonOption func(*Button)

// OnClick registers the click handler.
func OnClick(fn func()) ButtonOption {
	return func(b *Button) { b.onClick = fn }
}

// ButtonColor overrides the button's base fill color (RolePrimary).
func ButtonColor(c color.Color) ButtonOption {
	return func(b *Button) { b.SetColor(RolePrimary, c) }
}

// ButtonTextColor overrides the label color (RoleOnPrimary).
func ButtonTextColor(c color.Color) ButtonOption {
	return func(b *Button) { b.SetColor(RoleOnPrimary, c) }
}

// ButtonFont overrides the label font.
func ButtonFont(f render.FontFace) ButtonOption {
	return func(b *Button) { b.font = f }
}

// NewButton returns a Button showing text, configured by opts.
func NewButton(text string, opts ...ButtonOption) *Button {
	b := &Button{BaseWidget: NewBase(), text: text}
	for _, o := range opts {
		o(b)
	}
	return b
}

// SetText replaces the button's label and requests a re-layout.
func (b *Button) SetText(s string) {
	b.text = s
	b.Invalidate()
}

// SetOnClick replaces the click handler.
func (b *Button) SetOnClick(fn func()) { b.onClick = fn }

// SetFont overrides the button's label font (nil falls back to the theme font).
func (b *Button) SetFont(f render.FontFace) {
	b.font = f
	b.Invalidate()
}

func (b *Button) face() render.FontFace {
	if b.font != nil {
		return b.font
	}
	return b.appTheme().Font
}

// MinSize returns the label size plus the button's internal padding.
func (b *Button) MinSize() geom.Size {
	f := b.face()
	if f == nil {
		return geom.Size{}
	}
	s := f.Measure(b.text)
	return geom.Size{
		W: s.W + buttonPadding.Left + buttonPadding.Right,
		H: s.H + buttonPadding.Top + buttonPadding.Bottom,
	}
}

// fillColor resolves the background color for the button's current state.
func (b *Button) fillColor() color.Color {
	if !b.Enabled() {
		return b.ColorOf(RoleDisabled)
	}
	base := b.ColorOf(RolePrimary)
	switch {
	case b.pressed && b.hover:
		return darken(base, 0.8)
	case b.hover:
		return lighten(base, 1.15)
	default:
		return base
	}
}

func (b *Button) labelColor() color.Color {
	return b.ColorOf(RoleOnPrimary)
}

// Focusable reports whether the button can take keyboard focus (only when
// enabled).
func (b *Button) Focusable() bool { return b.Enabled() }

// Draw paints the background, a border, the centered label, and a focus ring
// when focused.
func (b *Button) Draw(c render.Canvas) {
	rect := b.Bounds()
	c.FillRect(rect, b.fillColor())
	c.StrokeRect(rect, b.ColorOf(RoleBorder), 1)
	if b.focused {
		ring := rect.Inset(geom.UniformInsets(2))
		c.StrokeRect(ring, b.ColorOf(RoleAccent), 1)
	}

	if f := b.face(); f != nil {
		size := f.Measure(b.text)
		x := rect.X + (rect.W-size.W)/2
		y := rect.Y + (rect.H-size.H)/2
		c.DrawText(b.text, geom.Point{X: x, Y: y}, f, b.labelColor())
	}
}

// activate invokes the click handler if one is registered.
func (b *Button) activate() {
	if b.onClick != nil {
		b.onClick()
	}
}

// HandleEvent updates hover/press/focus state and fires OnClick on a click or
// on Space/Enter while focused. The actual click is derived by the dispatcher
// (press and release on the same widget) and delivered as EventClick.
func (b *Button) HandleEvent(ev *Event) bool {
	if !b.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		b.hover = true
		return true
	case EventPointerLeave:
		b.hover = false
		return true
	case EventPointerDown:
		if ev.Button == render.MouseLeft {
			b.pressed = true
			return true
		}
	case EventPointerUp:
		if ev.Button == render.MouseLeft {
			b.pressed = false
			return true
		}
	case EventClick:
		b.activate()
		return true
	case EventFocusGained:
		b.focused = true
		return true
	case EventFocusLost:
		b.focused = false
		b.pressed = false
		return true
	case EventKeyDown:
		if ev.Key == render.KeySpace || ev.Key == render.KeyEnter {
			b.activate()
			return true
		}
	}
	return false
}

// darken returns c scaled toward black by factor (0..1).
func darken(c color.Color, factor float64) color.Color { return scaleRGB(c, factor) }

// lighten returns c scaled toward white-ish by factor (>1), clamped to 255.
func lighten(c color.Color, factor float64) color.Color { return scaleRGB(c, factor) }

func scaleRGB(c color.Color, factor float64) color.Color {
	r, g, bl, a := c.RGBA() // 16-bit premultiplied; here colors are opaque
	scale := func(v uint32) uint8 {
		f := float64(v>>8) * factor
		if f > 255 {
			f = 255
		}
		if f < 0 {
			f = 0
		}
		return uint8(f)
	}
	return color.RGBA{R: scale(r), G: scale(g), B: scale(bl), A: uint8(a >> 8)}
}
