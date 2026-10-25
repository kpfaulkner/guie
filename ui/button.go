package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// buttonPadding is the space reserved around a button's content, used by MinSize.
var buttonPadding = geom.Insets{Top: 6, Right: 14, Bottom: 6, Left: 14}

// buttonIconGap is the space between a button's icon and its label.
const buttonIconGap = 6

// Button is a clickable widget that draws a themed background and a centered
// label and/or icon, reacting to hover and press. Click handling is delivered
// through the pointer events dispatched by the App: a click fires when the
// pointer is released over the button after being pressed on it.
type Button struct {
	BaseWidget
	text    string
	icon    render.Image // optional; drawn before the label
	onClick func()

	hover   bool
	pressed bool
	focused bool
	flat    bool // no fill/border until hover/press (toolbar/ghost style)

	hoverFade fade // eases toward 1 while hovered
	pressFade fade // eases toward 1 while pressed over the button

	font render.FontFace
}

// ButtonOption configures a Button during NewButton.
type ButtonOption func(*Button)

// ButtonColour overrides the button's base fill colour (RolePrimary).
func ButtonColour(c color.Color) ButtonOption {
	return func(b *Button) { b.SetColour(RolePrimary, c) }
}

// ButtonTextColour overrides the label colour (RoleOnPrimary).
func ButtonTextColour(c color.Color) ButtonOption {
	return func(b *Button) { b.SetColour(RoleOnPrimary, c) }
}

// ButtonFont overrides the label font.
func ButtonFont(f render.FontFace) ButtonOption {
	return func(b *Button) { b.font = f }
}

// ButtonImage sets an icon drawn before the label. With an empty text the
// button shows just the image.
func ButtonImage(img render.Image) ButtonOption {
	return func(b *Button) { b.icon = img }
}

// ButtonFlat makes the button "flat": no fill or border until hovered/pressed.
// Useful for toolbars and link-style actions.
func ButtonFlat() ButtonOption {
	return func(b *Button) { b.flat = true }
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

// OnClick registers the handler invoked when the button is activated (click or
// Space/Enter while focused).
func (b *Button) OnClick(fn func()) { b.onClick = fn }

// SetFont overrides the button's label font (nil falls back to the theme font).
func (b *Button) SetFont(f render.FontFace) {
	b.font = f
	b.Invalidate()
}

// SetImage sets (or clears, with nil) the button's icon and requests a re-layout.
func (b *Button) SetImage(img render.Image) {
	b.icon = img
	b.Invalidate()
}

// contentSize returns the combined icon+label size (excluding padding).
func (b *Button) contentSize() geom.Size {
	var iconW, iconH float64
	if b.icon != nil {
		s := b.icon.Size()
		iconW, iconH = s.W, s.H
	}
	var textW, textH float64
	if b.text != "" {
		if f := b.face(); f != nil {
			s := f.Measure(b.text)
			textW, textH = s.W, s.H
		}
	}
	gap := 0.0
	if iconW > 0 && textW > 0 {
		gap = buttonIconGap
	}
	return geom.Size{W: iconW + gap + textW, H: maxF(iconH, textH)}
}

func (b *Button) face() render.FontFace {
	if b.font != nil {
		return b.font
	}
	return b.appTheme().Font
}

// MinSize returns the icon+label size plus the button's internal padding.
func (b *Button) MinSize() geom.Size {
	c := b.contentSize()
	return geom.Size{
		W: c.W + buttonPadding.Left + buttonPadding.Right,
		H: c.H + buttonPadding.Top + buttonPadding.Bottom,
	}
}

// fillColour resolves the background colour for the button's current state.
func (b *Button) fillColour() color.Color {
	if !b.Enabled() {
		return b.ColourOf(RoleDisabled)
	}
	base := b.ColourOf(RolePrimary)
	// Cross-fade idle -> hover -> pressed rather than switching, so a state
	// change eases in over transitionDuration.
	c := mixColour(base, lighten(base, 1.15), b.hoverFade.level)
	return mixColour(c, darken(base, 0.8), b.pressFade.level)
}

// syncFades points the hover and press fades at the button's current state.
// Called after every event, so any path that changes hover/pressed is covered.
func (b *Button) syncFades() {
	hv, pr := 0.0, 0.0
	if b.hover {
		hv = 1
	}
	if b.pressed && b.hover {
		pr = 1
	}
	b.fadeTo(&b.hoverFade, hv)
	b.fadeTo(&b.pressFade, pr)
}

func (b *Button) labelColour() color.Color {
	switch {
	case !b.Enabled():
		return b.ColourOf(RoleTextMuted)
	case b.flat:
		return b.ColourOf(RoleText)
	default:
		return b.ColourOf(RoleOnPrimary)
	}
}

// flatHighlight is the subtle background a flat button shows on hover/press.
func (b *Button) flatHighlight() color.Color {
	base := b.ColourOf(RoleSurface)
	if b.pressed && b.hover {
		return tint(base, tintPress)
	}
	return tint(base, tintFlat)
}

// flatFill is the flat button's current highlight: the hover colour faded in by
// the hover level, shifting to the pressed colour as the press level rises.
// Fully transparent at rest.
func (b *Button) flatFill() color.Color {
	base := b.ColourOf(RoleSurface)
	c := mixColour(tint(base, tintFlat), tint(base, tintPress), b.pressFade.level)
	return withAlpha(c, b.hoverFade.level)
}

// Focusable reports whether the button can take keyboard focus (only when
// enabled).
func (b *Button) Focusable() bool { return b.Enabled() }

// Draw paints the background, a border, the centered icon and/or label, and a
// focus ring when focused.
func (b *Button) Draw(c render.Canvas) {
	rect := b.Bounds()
	rad := b.cornerRadius()
	if b.flat {
		if b.Enabled() && b.hoverFade.level > 0 {
			c.FillRoundRect(rect, rad, b.flatFill())
		}
	} else {
		c.FillRoundRect(rect, rad, b.fillColour())
		strokeFocusable(c, rect, rad, b.ColourOf(RoleBorder), b.ColourOf(RoleAccent), b.focused)
	}
	if b.flat && b.focused {
		// A flat button has no border at rest, only the focus ring.
		strokeFocusable(c, rect, rad, b.ColourOf(RoleBorder), b.ColourOf(RoleAccent), true)
	}

	content := b.contentSize()
	x := rect.X + (rect.W-content.W)/2 // left edge of the icon+label group

	if b.icon != nil {
		s := b.icon.Size()
		dst := geom.Rect{X: x, Y: rect.Y + (rect.H-s.H)/2, W: s.W, H: s.H}
		c.DrawImage(b.icon, dst)
		x += s.W
		if b.text != "" {
			x += buttonIconGap
		}
	}

	if b.text != "" {
		if f := b.face(); f != nil {
			c.DrawText(b.text, geom.Point{X: x, Y: vCenterY(f, rect.Y, rect.H)}, f, b.labelColour())
		}
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
	defer b.syncFades()
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
	r, g, bl, a := c.RGBA() // 16-bit premultiplied; here colours are opaque
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
