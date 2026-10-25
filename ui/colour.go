package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/theme"
)

// ColourRole names a semantic colour slot, mirroring the theme Palette. Every
// widget resolves the colours it draws through these roles, so any role can be
// overridden per widget while unset roles fall back to the theme.
type ColourRole int

const (
	RoleBackground ColourRole = iota
	RoleSurface
	RolePrimary
	RoleOnPrimary
	RoleText
	RoleTextMuted
	RoleBorder
	RoleAccent
	RoleDisabled
	// Status colours and the modal backdrop, added after the first nine so existing
	// values do not move.
	RoleSuccess
	RoleWarning
	RoleError
	RoleInfo
	RoleScrim
)

// paletteColour returns the palette colour for a role.
func paletteColour(p theme.Palette, role ColourRole) color.Color {
	switch role {
	case RoleBackground:
		return p.Background
	case RoleSurface:
		return p.Surface
	case RolePrimary:
		return p.Primary
	case RoleOnPrimary:
		return p.OnPrimary
	case RoleText:
		return p.Text
	case RoleTextMuted:
		return p.TextMuted
	case RoleBorder:
		return p.Border
	case RoleAccent:
		return p.Accent
	case RoleDisabled:
		return p.Disabled
	case RoleSuccess:
		return orColour(p.Success, theme.DefaultPalette().Success)
	case RoleWarning:
		return orColour(p.Warning, theme.DefaultPalette().Warning)
	case RoleError:
		return orColour(p.Error, theme.DefaultPalette().Error)
	case RoleInfo:
		return orColour(p.Info, p.Primary) // info toasts have always followed Primary
	case RoleScrim:
		return orColour(p.Scrim, theme.DefaultPalette().Scrim)
	default:
		return p.Text
	}
}

// SetColour overrides a colour role for this widget. Passing nil clears the
// override (the role falls back to the theme again). It requests a redraw.
func (b *BaseWidget) SetColour(role ColourRole, c color.Color) {
	if c == nil {
		delete(b.colours, role)
	} else {
		if b.colours == nil {
			b.colours = make(map[ColourRole]color.Color)
		}
		b.colours[role] = c
	}
	b.Invalidate()
}

// ColourOf returns the effective colour for a role: the per-widget override if
// set, otherwise the active theme's palette colour. It is safe before mount
// (falls back to the default theme).
func (b *BaseWidget) ColourOf(role ColourRole) color.Color {
	if b.colours != nil {
		if c, ok := b.colours[role]; ok {
			return c
		}
	}
	return paletteColour(b.appTheme().Palette, role)
}

// tintKind names a state shade. The amounts come from the theme (Theme.States), so
// a theme can make hover and pressed more or less obvious; see tintOf.
type tintKind int

const (
	tintHeader tintKind = iota // table header band (striping uses half)
	tintHover                  // hovered row, cell or tab
	tintStrong                 // stronger emphasis: hovered header cell, tooltip body
	tintFlat                   // flat button hover
	tintPress                  // flat button pressed
)

// flatHoverScale makes a flat button's hover a little stronger than a list row's: the
// old constants were 0.08 and 0.06.
const flatHoverScale = 4.0 / 3.0

// tintAmount resolves a tint kind against a theme's state tints.
func tintAmount(s theme.StateTints, k tintKind) float64 {
	switch k {
	case tintHeader:
		return s.Header
	case tintHover:
		return s.Hover
	case tintStrong:
		return s.Emphasis
	case tintFlat:
		return s.Hover * flatHoverScale
	case tintPress:
		return s.Pressed
	}
	return 0
}

// tintOf is how far this widget shifts a base colour for a state, per the theme.
func (b *BaseWidget) tintOf(k tintKind) float64 { return tintAmount(b.appTheme().States, k) }

// borderW is the thickness of control outlines; focusW the keyboard focus ring.
func (b *BaseWidget) borderW() float64 { return b.appTheme().BorderWidth }
func (b *BaseWidget) focusW() float64  { return b.appTheme().FocusWidth }

// tint nudges c away from the page: toward white if c is dark, toward black if
// it is light. Scaling a colour up (lighten) does nothing on a light theme,
// where surfaces are already near white, so hover/raised shades go through here
// to stay visible on both.
func tint(c color.Color, t float64) color.Color {
	r, g, b, _ := c.RGBA()
	lum := (299*r + 587*g + 114*b) / 1000
	if lum < 0x8000 {
		return mixColour(c, color.White, t)
	}
	return mixColour(c, color.Black, t)
}

// orColour returns c, or def if c is nil, for palette entries a hand-built theme
// may not set (an App's theme is always completed, but a widget can be drawn before
// it is mounted).
func orColour(c, def color.Color) color.Color {
	if c != nil {
		return c
	}
	return def
}

// densityOf is a theme's Density, or 1 for a theme that has none set.
func densityOf(th theme.Theme) float64 {
	if th.Density <= 0 {
		return 1
	}
	return th.Density
}

// density is the active theme's Density. It reads the theme through the tree
// context rather than copying it, because it is asked for on every layout and
// every hit test.
func (b *BaseWidget) density() float64 {
	if b.ctx != nil && b.ctx.theme != nil {
		return densityOf(*b.ctx.theme)
	}
	return 1
}

// px scales a padding or gap by the theme's Density. At the default density of 1 it
// returns v exactly, so the default look is unchanged to the bit. Only the space
// around content is scaled, never text or control glyphs.
func (b *BaseWidget) px(v float64) float64 { return v * b.density() }

// insets scales every side of a padding by the theme's Density.
func (b *BaseWidget) insets(in geom.Insets) geom.Insets {
	d := b.density()
	return geom.Insets{Top: in.Top * d, Right: in.Right * d, Bottom: in.Bottom * d, Left: in.Left * d}
}

// px is the App's own (for toasts and tooltips, which are not widgets).
func (a *App) px(v float64) float64 { return v * densityOf(a.theme) }
