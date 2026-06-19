package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const progressHeight = 14

// ProgressBar is a non-interactive widget that fills horizontally in proportion
// to its value in [0,1].
type ProgressBar struct {
	BaseWidget
	value float64
}

// NewProgressBar returns a ProgressBar starting at value v (clamped to [0,1]).
func NewProgressBar(v float64) *ProgressBar {
	return &ProgressBar{BaseWidget: NewBase(), value: clamp01(v)}
}

// Value returns the current value in [0,1].
func (p *ProgressBar) Value() float64 { return p.value }

// SetValue sets the fill proportion (clamped to [0,1]).
func (p *ProgressBar) SetValue(v float64) {
	p.value = clamp01(v)
	p.Invalidate()
}

// MinSize returns a default width and fixed height.
func (p *ProgressBar) MinSize() geom.Size { return geom.Size{W: 120, H: progressHeight} }

// Draw renders the track and the filled portion.
func (p *ProgressBar) Draw(canvas render.Canvas) {
	pal := p.appTheme().Palette
	b := p.Bounds()
	canvas.FillRect(b, pal.Surface)
	if p.value > 0 {
		fill := geom.Rect{X: b.X, Y: b.Y, W: b.W * p.value, H: b.H}
		canvas.FillRect(fill, pal.Primary)
	}
	canvas.StrokeRect(b, pal.Border, 1)
}
