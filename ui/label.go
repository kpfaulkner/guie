package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Label is a non-interactive widget that displays a line of text.
//
// Rendering currently uses Ebiten's built-in debug font via
// ebitenutil.DebugPrintAt, which is monochrome and fixed-size. A production
// implementation would swap this for a text/v2 face to support custom fonts,
// sizing, and color.
type Label struct {
	BaseWidget
	Text string
}

// NewLabel returns a label occupying r with the given text.
func NewLabel(r Rect, text string) *Label {
	return &Label{BaseWidget: NewBase(r), Text: text}
}

// Draw renders the label's text at its top-left corner.
func (l *Label) Draw(dst *ebiten.Image, origin Point) {
	abs := l.bounds.Add(origin)
	ebitenutil.DebugPrintAt(dst, l.Text, abs.X, abs.Y)
}
