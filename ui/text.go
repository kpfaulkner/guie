package ui

import (
	"image/color"
	"strings"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// vCenterY returns the y at which to draw a single line of text so its glyph
// block (ascent + descent, ignoring line gap) is optically centered within the
// vertical span [rectY, rectY+rectH]. Text is drawn from its top, so callers
// pass the result as the text origin's Y.
func vCenterY(f render.FontFace, rectY, rectH float64) float64 {
	m := f.Metrics()
	return rectY + (rectH-(m.Ascent+m.Descent))/2
}

// drawText renders s within rect: each newline-separated line is aligned
// horizontally per align, and the whole block is centered vertically. Lines are
// positioned individually (one DrawText per line) rather than passing embedded
// newlines through to the backend, so multi-line text never overlaps regardless
// of backend newline handling. For single-line text this matches vCenterY.
func drawText(c render.Canvas, s string, rect geom.Rect, align geom.Alignment, f render.FontFace, col color.Color) {
	m := f.Metrics()
	lines := strings.Split(s, "\n")
	blockH := m.LineHeight*float64(len(lines)-1) + (m.Ascent + m.Descent)
	startY := rect.Y + (rect.H-blockH)/2
	for i, line := range lines {
		x, _ := alignSpan(align, rect.X, rect.W, f.Measure(line).W)
		c.DrawText(line, geom.Point{X: x, Y: startY + m.LineHeight*float64(i)}, f, col)
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
