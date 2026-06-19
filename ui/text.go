package ui

import "github.com/kpfaulkner/uiframework/render"

// vCenterY returns the y at which to draw a single line of text so its glyph
// block (ascent + descent, ignoring line gap) is optically centered within the
// vertical span [rectY, rectY+rectH]. Text is drawn from its top, so callers
// pass the result as the text origin's Y.
func vCenterY(f render.FontFace, rectY, rectH float64) float64 {
	m := f.Metrics()
	return rectY + (rectH-(m.Ascent+m.Descent))/2
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
