package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// Soft-shadow parameters. The shadow is faked by stacking several translucent,
// progressively larger rounded rectangles offset downward: overlapping builds
// up opacity toward the center, giving a soft falloff without a real blur.
const (
	shadowLayers  = 8
	shadowSpread  = 2.5 // how much each successive layer grows, in pixels
	shadowOffsetY = 4   // downward offset (light comes from above)
	shadowAlpha   = 30  // per-layer alpha; layers compound to a darker halo
)

// drawShadow paints a soft drop shadow beneath a rounded rect of the given
// bounds and corner radius.
func drawShadow(c render.Canvas, bounds geom.Rect, radius float64) {
	for i := shadowLayers; i >= 1; i-- {
		grow := float64(i) * shadowSpread
		r := geom.Rect{
			X: bounds.X - grow,
			Y: bounds.Y - grow + shadowOffsetY,
			W: bounds.W + 2*grow,
			H: bounds.H + 2*grow,
		}
		c.FillRoundRect(r, radius+grow, color.RGBA{A: shadowAlpha})
	}
}
