package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// strokeFocusable outlines a control's rounded rectangle. Unfocused it is the
// theme's border (BorderWidth, in the Border colour). Focused it is the theme's
// focus ring (FocusWidth, in the Accent colour) drawn just inside the bounds, so a
// parent's clip never trims it, and covering the border. Every focusable control
// goes through this, so keyboard focus looks the same everywhere. radius 0 gives
// square corners.
//
// The ring has a halo: a thin line of the page colour just inside it. Without it a
// ring the colour of the control's own fill (the light theme's Accent is its
// Primary) vanishes on a filled button; with it the ring is always bounded by
// something that contrasts with it on one side and the page on the other.
func (b *BaseWidget) strokeFocusable(c render.Canvas, r geom.Rect, radius float64, focused bool) {
	if !focused {
		c.StrokeRoundRect(r, radius, b.ColourOf(RoleBorder), b.borderW())
		return
	}
	fw := b.focusW()
	hw := haloWidth(b.borderW())
	c.StrokeRoundRect(r.Inset(geom.UniformInsets(fw/2)), maxF(0, radius-fw/2), b.ColourOf(RoleAccent), fw)
	c.StrokeRoundRect(r.Inset(geom.UniformInsets(fw+hw/2)), maxF(0, radius-fw-hw/2), b.ColourOf(RoleBackground), hw)
}

// strokeFocusableCircle is strokeFocusable for a circular control (radio).
func (b *BaseWidget) strokeFocusableCircle(c render.Canvas, center geom.Point, radius float64, focused bool) {
	if !focused {
		c.StrokeCircle(center, radius, b.ColourOf(RoleBorder), b.borderW())
		return
	}
	fw := b.focusW()
	hw := haloWidth(b.borderW())
	c.StrokeCircle(center, maxF(0, radius-fw/2), b.ColourOf(RoleAccent), fw)
	c.StrokeCircle(center, maxF(0, radius-fw-hw/2), b.ColourOf(RoleBackground), hw)
}

// haloWidth is the halo's thickness: as thick as the border, so a theme with heavy
// outlines gets a halo that matches, and never under a pixel.
func haloWidth(border float64) float64 { return maxF(1, border) }
