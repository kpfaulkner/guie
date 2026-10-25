package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// focusRingWidth is the thickness of the accent ring on a focused control.
const focusRingWidth = 2

// strokeFocusable outlines a control's rounded rectangle. Unfocused it is the
// usual 1px border; focused it is a focusRingWidth accent ring drawn just inside
// the bounds (so a parent's clip never trims it) and covering the border. Every
// focusable control goes through this, so keyboard focus looks the same
// everywhere. radius 0 gives square corners.
func strokeFocusable(c render.Canvas, r geom.Rect, radius float64, border, accent color.Color, focused bool) {
	if !focused {
		c.StrokeRoundRect(r, radius, border, 1)
		return
	}
	c.StrokeRoundRect(r.Inset(geom.UniformInsets(focusRingWidth/2)), maxF(0, radius-focusRingWidth/2), accent, focusRingWidth)
}

// strokeFocusableCircle is strokeFocusable for a circular control (radio).
func strokeFocusableCircle(c render.Canvas, center geom.Point, radius float64, border, accent color.Color, focused bool) {
	if !focused {
		c.StrokeCircle(center, radius, border, 1)
		return
	}
	c.StrokeCircle(center, maxF(0, radius-focusRingWidth/2), accent, focusRingWidth)
}
