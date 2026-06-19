package ui

import (
	"math"

	"github.com/kpfaulkner/uiframework/geom"
)

// Stack overlays all children in the same area, in add order (later children
// draw on top). Each child is placed within the content rectangle per its Align
// on both axes (AlignStretch fills). It is the basis for dialogs, popups and
// centering a single child.
type Stack struct{}

// NewStack returns a Stack layout.
func NewStack() *Stack { return &Stack{} }

// Measure returns the size of the largest child on each axis.
func (s *Stack) Measure(items []Item) geom.Size {
	var w, h float64
	for _, it := range items {
		m := it.Widget.MinSize()
		w = math.Max(w, m.W)
		h = math.Max(h, m.H)
	}
	return geom.Size{W: w, H: h}
}

// Arrange places each child within the content rectangle per its Align.
func (s *Stack) Arrange(items []Item, content geom.Rect) {
	for _, it := range items {
		it.Widget.SetBounds(alignInCell(it.Data.Align, content, it.Widget.MinSize()))
	}
}
