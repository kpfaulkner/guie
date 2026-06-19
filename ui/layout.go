package ui

import "github.com/kpfaulkner/guie/geom"

// Layout positions a container's children. A container delegates measuring and
// arranging to its Layout; without one it falls back to leaving children at
// their assigned bounds.
//
// Measure returns the minimum content size (excluding the container's own
// padding) needed to lay the items out. Arrange assigns each item's Bounds
// within the given content rectangle (already inset by padding).
type Layout interface {
	Measure(items []Item) geom.Size
	Arrange(items []Item, content geom.Rect)
}

// Item pairs a child widget with the per-child layout parameters its container
// recorded when the child was added.
type Item struct {
	Widget Widget
	Data   LayoutData
}

// LayoutData holds per-child layout parameters. Different layouts interpret the
// fields they care about and ignore the rest.
type LayoutData struct {
	// Weight is the child's share of leftover space along a Box's main axis (or
	// a track's weight in a Grid). 0 means "use the child's minimum size".
	Weight int
	// Align controls the child's alignment within the space the layout gives it
	// (the cross axis in a Box, the whole cell in a Grid/Stack). AlignStretch
	// fills the space.
	Align geom.Alignment
	// ColSpan and RowSpan are the number of grid cells a child occupies. Used
	// only by Grid; both default to 1.
	ColSpan, RowSpan int
}

func defaultLayoutData() LayoutData {
	return LayoutData{Align: geom.AlignStretch, ColSpan: 1, RowSpan: 1}
}

// ItemOption configures a child's LayoutData when it is added to a container.
type ItemOption func(*LayoutData)

// Weight sets the child's stretch weight along the layout's main axis.
func Weight(n int) ItemOption {
	if n < 0 {
		n = 0
	}
	return func(d *LayoutData) { d.Weight = n }
}

// Align sets the child's alignment within the space its layout assigns it.
func Align(a geom.Alignment) ItemOption {
	return func(d *LayoutData) { d.Align = a }
}

// Span sets how many grid columns and rows a child occupies (Grid only). Values
// below 1 are clamped to 1.
func Span(cols, rows int) ItemOption {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return func(d *LayoutData) { d.ColSpan, d.RowSpan = cols, rows }
}

// --- shared axis helpers used by the layout implementations ---

// mainExtent returns the size component along dir.
func mainExtent(dir geom.Direction, s geom.Size) float64 {
	if dir == geom.Horizontal {
		return s.W
	}
	return s.H
}

// crossExtent returns the size component across dir.
func crossExtent(dir geom.Direction, s geom.Size) float64 {
	if dir == geom.Horizontal {
		return s.H
	}
	return s.W
}

// sizeFor builds a Size from main/cross extents for the given direction.
func sizeFor(dir geom.Direction, main, cross float64) geom.Size {
	if dir == geom.Horizontal {
		return geom.Size{W: main, H: cross}
	}
	return geom.Size{W: cross, H: main}
}

// alignSpan resolves a child's position and length within an available span,
// given its preferred size and alignment.
func alignSpan(a geom.Alignment, start, available, preferred float64) (pos, length float64) {
	switch a {
	case geom.AlignStretch:
		return start, available
	case geom.AlignCenter:
		return start + (available-preferred)/2, preferred
	case geom.AlignEnd:
		return start + available - preferred, preferred
	default: // AlignStart
		return start, preferred
	}
}

// distributeTracks splits total (minus inter-track spacing) into n track sizes
// by weight. A nil/short weights slice defaults missing weights to 1, yielding
// equal tracks.
func distributeTracks(n int, total, spacing float64, weights []int) []float64 {
	tracks := make([]float64, n)
	if n == 0 {
		return tracks
	}
	avail := total - spacing*float64(n-1)
	if avail < 0 {
		avail = 0
	}
	totalWeight := 0
	for i := 0; i < n; i++ {
		w := 1
		if i < len(weights) {
			w = weights[i]
			if w < 0 {
				w = 0
			}
		}
		tracks[i] = float64(w)
		totalWeight += w
	}
	if totalWeight == 0 {
		return tracks // all zero-weight tracks collapse to 0
	}
	for i := range tracks {
		tracks[i] = avail * tracks[i] / float64(totalWeight)
	}
	return tracks
}
