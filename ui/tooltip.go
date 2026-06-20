package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// tooltipDelayTicks is how long the pointer must rest before a tooltip appears.
// Update runs at a fixed tick rate (60/sec by default), so this is ~0.5s.
const tooltipDelayTicks = 30

// tooltip layout constants.
const (
	tooltipPad       = 6
	tooltipCursorOff = 16 // offset from the cursor so the tip doesn't sit under it
)

// updateTooltip advances the hover timer. The tooltip appears once the pointer
// has rested (unmoved) over a widget with tooltip text for tooltipDelayTicks;
// any movement hides it and restarts the timer.
func (a *App) updateTooltip(pos geom.Point) {
	if pos != a.lastPointer {
		a.lastPointer = pos
		a.hideTooltip()
		return
	}
	if a.tooltipText != "" {
		return // already showing
	}
	a.tooltipTicks++
	if a.tooltipTicks >= tooltipDelayTicks && a.hovered != nil {
		if tip := a.hovered.Tooltip(); tip != "" {
			a.tooltipText = tip
			a.tooltipPos = pos
		}
	}
}

func (a *App) hideTooltip() {
	a.tooltipTicks = 0
	a.tooltipText = ""
}

// drawTooltip paints the active tooltip near the cursor, clamped to the surface.
func (a *App) drawTooltip(c render.Canvas) {
	if a.tooltipText == "" {
		return
	}
	f := a.theme.Font
	if f == nil {
		return
	}
	pal := a.theme.Palette

	size := f.Measure(a.tooltipText)
	w := size.W + 2*tooltipPad
	h := size.H + 2*tooltipPad
	x := a.tooltipPos.X + tooltipCursorOff
	y := a.tooltipPos.Y + tooltipCursorOff

	// Keep the tooltip on screen.
	if x+w > a.surfaceSize.W {
		x = a.surfaceSize.W - w
	}
	if y+h > a.surfaceSize.H {
		y = a.surfaceSize.H - h
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	rect := geom.Rect{X: x, Y: y, W: w, H: h}
	rad := a.theme.CornerRadius
	if a.shadows {
		drawShadow(c, rect, rad)
	}
	c.FillRoundRect(rect, rad, lighten(pal.Surface, 1.3))
	c.StrokeRoundRect(rect, rad, pal.Border, 1)
	inner := geom.Rect{X: x + tooltipPad, Y: y + tooltipPad, W: size.W, H: size.H}
	drawText(c, a.tooltipText, inner, geom.AlignStart, f, pal.Text)
}
