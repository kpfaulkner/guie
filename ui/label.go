package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// Label is a non-interactive widget that draws a single line of text. Color and
// font fall back to the theme when not overridden; the text is aligned
// horizontally per Align and centered vertically within the widget's bounds.
type Label struct {
	BaseWidget
	text  string
	font  render.FontFace // nil → theme font
	align geom.Alignment  // horizontal alignment within bounds
}

// LabelOption configures a Label during NewLabel.
type LabelOption func(*Label)

// LabelColor overrides the text color (RoleText).
func LabelColor(c color.Color) LabelOption {
	return func(l *Label) { l.SetColor(RoleText, c) }
}

// LabelFont overrides the font face.
func LabelFont(f render.FontFace) LabelOption {
	return func(l *Label) { l.font = f }
}

// LabelAlign sets the horizontal text alignment within the label's bounds.
func LabelAlign(a geom.Alignment) LabelOption {
	return func(l *Label) { l.align = a }
}

// NewLabel returns a Label showing text, configured by opts.
func NewLabel(text string, opts ...LabelOption) *Label {
	l := &Label{BaseWidget: NewBase(), text: text, align: geom.AlignStart}
	for _, o := range opts {
		o(l)
	}
	return l
}

// SetText replaces the label's text and requests a re-layout (its size changes).
func (l *Label) SetText(s string) {
	l.text = s
	l.Invalidate()
}

// SetFont overrides the label's font face (nil falls back to the theme font).
func (l *Label) SetFont(f render.FontFace) {
	l.font = f
	l.Invalidate()
}

// Text returns the label's current text.
func (l *Label) Text() string { return l.text }

func (l *Label) face() render.FontFace {
	if l.font != nil {
		return l.font
	}
	return l.appTheme().Font
}

// MinSize returns the measured size of the text.
func (l *Label) MinSize() geom.Size {
	f := l.face()
	if f == nil {
		return geom.Size{}
	}
	return f.Measure(l.text)
}

// Draw renders the text aligned within the label's bounds.
func (l *Label) Draw(c render.Canvas) {
	f := l.face()
	if f == nil {
		return
	}
	b := l.Bounds()
	size := f.Measure(l.text)
	x, _ := alignSpan(l.align, b.X, b.W, size.W)
	c.DrawText(l.text, geom.Point{X: x, Y: vCenterY(f, b.Y, b.H)}, f, l.ColorOf(RoleText))
}
