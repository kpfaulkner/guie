package ui

import (
	"image/color"

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
