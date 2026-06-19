package ui

import (
	"math"

	"github.com/kpfaulkner/guie/geom"
)

// Box arranges children in a single row (Horizontal) or column (Vertical).
//
// Children are laid out sequentially along the main axis with Spacing between
// them. Each child takes its minimum main-axis size; any leftover main-axis
// space is distributed among children in proportion to their Weight (a child
// with Weight 0 stays at its minimum). On the cross axis each child is placed
// per its Align (AlignStretch fills the box). Box with weights is also the
// framework's "flex" layout.
type Box struct {
	Direction geom.Direction
	Spacing   float64
}

// NewBox returns a Box arranging children along dir.
func NewBox(dir geom.Direction) *Box { return &Box{Direction: dir} }

// HBox returns a horizontal Box with the given spacing.
func HBox(spacing float64) *Box { return &Box{Direction: geom.Horizontal, Spacing: spacing} }

// VBox returns a vertical Box with the given spacing.
func VBox(spacing float64) *Box { return &Box{Direction: geom.Vertical, Spacing: spacing} }

// Measure returns the minimum size: children stacked along the main axis with
// spacing, and the widest child on the cross axis.
func (b *Box) Measure(items []Item) geom.Size {
	if len(items) == 0 {
		return geom.Size{}
	}
	var main, cross float64
	for _, it := range items {
		m := it.Widget.MinSize()
		main += mainExtent(b.Direction, m)
		cross = math.Max(cross, crossExtent(b.Direction, m))
	}
	main += b.Spacing * float64(len(items)-1)
	return sizeFor(b.Direction, main, cross)
}

// Arrange positions each child along the main axis, distributing leftover space
// by weight, and aligns each child on the cross axis.
func (b *Box) Arrange(items []Item, content geom.Rect) {
	n := len(items)
	if n == 0 {
		return
	}

	mainAvail := mainExtent(b.Direction, content.Size())
	crossAvail := crossExtent(b.Direction, content.Size())

	// Resolve each child's main-axis size: minimum plus a weighted share of the
	// leftover space.
	mains := make([]float64, n)
	var totalMin float64
	totalWeight := 0
	for i, it := range items {
		mains[i] = mainExtent(b.Direction, it.Widget.MinSize())
		totalMin += mains[i]
		totalWeight += it.Data.Weight
	}
	free := mainAvail - totalMin - b.Spacing*float64(n-1)
	if free > 0 && totalWeight > 0 {
		for i, it := range items {
			if it.Data.Weight > 0 {
				mains[i] += free * float64(it.Data.Weight) / float64(totalWeight)
			}
		}
	}

	// Main-axis origin and cross-axis origin within the content rectangle.
	var mainPos, crossStart float64
	if b.Direction == geom.Horizontal {
		mainPos, crossStart = content.X, content.Y
	} else {
		mainPos, crossStart = content.Y, content.X
	}

	for i, it := range items {
		childCross := crossExtent(b.Direction, it.Widget.MinSize())
		crossPos, crossLen := alignSpan(it.Data.Align, crossStart, crossAvail, childCross)

		var r geom.Rect
		if b.Direction == geom.Horizontal {
			r = geom.Rect{X: mainPos, Y: crossPos, W: mains[i], H: crossLen}
		} else {
			r = geom.Rect{X: crossPos, Y: mainPos, W: crossLen, H: mains[i]}
		}
		it.Widget.SetBounds(r)
		mainPos += mains[i] + b.Spacing
	}
}
