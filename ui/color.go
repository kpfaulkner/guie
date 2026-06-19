package ui

import (
	"image/color"

	"github.com/kpfaulkner/uiframework/theme"
)

// ColorRole names a semantic color slot, mirroring the theme Palette. Every
// widget resolves the colors it draws through these roles, so any role can be
// overridden per widget while unset roles fall back to the theme.
type ColorRole int

const (
	RoleBackground ColorRole = iota
	RoleSurface
	RolePrimary
	RoleOnPrimary
	RoleText
	RoleTextMuted
	RoleBorder
	RoleAccent
	RoleDisabled
)

// paletteColor returns the palette color for a role.
func paletteColor(p theme.Palette, role ColorRole) color.Color {
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

// SetColor overrides a color role for this widget. Passing nil clears the
// override (the role falls back to the theme again). It requests a redraw.
func (b *BaseWidget) SetColor(role ColorRole, c color.Color) {
	if c == nil {
		delete(b.colors, role)
	} else {
		if b.colors == nil {
			b.colors = make(map[ColorRole]color.Color)
		}
		b.colors[role] = c
	}
	b.Invalidate()
}

// ColorOf returns the effective color for a role: the per-widget override if
// set, otherwise the active theme's palette color. It is safe before mount
// (falls back to the default theme).
func (b *BaseWidget) ColorOf(role ColorRole) color.Color {
	if b.colors != nil {
		if c, ok := b.colors[role]; ok {
			return c
		}
	}
	return paletteColor(b.appTheme().Palette, role)
}
