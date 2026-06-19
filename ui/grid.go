package ui

import (
	"math"

	"github.com/kpfaulkner/guie/geom"
)

// Grid arranges children into a fixed number of columns, flowing left to right
// and wrapping to a new row.
//
// A child may span multiple columns and/or rows (LayoutData.ColSpan/RowSpan, set
// with the Span item option); auto-flow skips cells already covered by a
// spanning child. Columns and rows fill the available content rectangle: column
// widths are distributed by ColWeights (equal if unset) and rows share the
// height by RowWeights (equal if unset). Each child is placed within its cell
// block per its Align (AlignStretch fills it).
type Grid struct {
	Columns    int
	Spacing    float64
	ColWeights []int
	RowWeights []int
}

// NewGrid returns a Grid with the given column count and spacing.
func NewGrid(columns int, spacing float64) *Grid {
	if columns < 1 {
		columns = 1
	}
	return &Grid{Columns: columns, Spacing: spacing}
}

func (g *Grid) cols() int {
	if g.Columns < 1 {
		return 1
	}
	return g.Columns
}

// cellPlacement is where an item sits in the grid and how many cells it covers.
type cellPlacement struct {
	row, col, colSpan, rowSpan int
}

// place assigns each item a cell block via sparse auto-flow, honouring spans,
// and returns the placements and the total number of rows used.
func (g *Grid) place(items []Item) ([]cellPlacement, int) {
	cols := g.cols()
	occupied := map[[2]int]bool{}
	free := func(r, c, cs, rs int) bool {
		if c+cs > cols {
			return false
		}
		for dr := 0; dr < rs; dr++ {
			for dc := 0; dc < cs; dc++ {
				if occupied[[2]int{r + dr, c + dc}] {
					return false
				}
			}
		}
		return true
	}

	placements := make([]cellPlacement, len(items))
	rows := 0
	cr, cc := 0, 0 // cursor
	for i, it := range items {
		cs, rs := spanOf(it.Data)
		if cs > cols {
			cs = cols
		}
		// Find the next free block at or after the cursor.
		r, c := cr, cc
		for {
			if c+cs > cols {
				c = 0
				r++
				continue
			}
			if free(r, c, cs, rs) {
				break
			}
			c++
		}
		for dr := 0; dr < rs; dr++ {
			for dc := 0; dc < cs; dc++ {
				occupied[[2]int{r + dr, c + dc}] = true
			}
		}
		placements[i] = cellPlacement{row: r, col: c, colSpan: cs, rowSpan: rs}
		if r+rs > rows {
			rows = r + rs
		}
		// Advance the cursor past the item.
		cr, cc = r, c+cs
		if cc >= cols {
			cr, cc = r+1, 0
		}
	}
	return placements, rows
}

func spanOf(d LayoutData) (cs, rs int) {
	cs, rs = d.ColSpan, d.RowSpan
	if cs < 1 {
		cs = 1
	}
	if rs < 1 {
		rs = 1
	}
	return
}

// Measure returns the minimum size needed for the placed grid, distributing a
// spanning child's minimum across the tracks it covers.
func (g *Grid) Measure(items []Item) geom.Size {
	if len(items) == 0 {
		return geom.Size{}
	}
	cols := g.cols()
	placements, rows := g.place(items)

	colMin := make([]float64, cols)
	rowMin := make([]float64, rows)
	for i, it := range items {
		p := placements[i]
		m := it.Widget.MinSize()
		wPer := m.W / float64(p.colSpan)
		hPer := m.H / float64(p.rowSpan)
		for dc := 0; dc < p.colSpan; dc++ {
			colMin[p.col+dc] = math.Max(colMin[p.col+dc], wPer)
		}
		for dr := 0; dr < p.rowSpan; dr++ {
			rowMin[p.row+dr] = math.Max(rowMin[p.row+dr], hPer)
		}
	}

	var w, h float64
	for _, cw := range colMin {
		w += cw
	}
	for _, rh := range rowMin {
		h += rh
	}
	w += g.Spacing * float64(cols-1)
	h += g.Spacing * float64(rows-1)
	return geom.Size{W: w, H: h}
}

// Arrange splits the content rectangle into a grid and places each child over
// its (possibly spanning) cell block.
func (g *Grid) Arrange(items []Item, content geom.Rect) {
	if len(items) == 0 {
		return
	}
	cols := g.cols()
	placements, rows := g.place(items)

	colW := distributeTracks(cols, content.W, g.Spacing, g.ColWeights)
	rowH := distributeTracks(rows, content.H, g.Spacing, g.RowWeights)
	colX := trackOrigins(colW, content.X, g.Spacing)
	rowY := trackOrigins(rowH, content.Y, g.Spacing)

	for i, it := range items {
		p := placements[i]
		cell := geom.Rect{
			X: colX[p.col],
			Y: rowY[p.row],
			W: spanExtent(colW, p.col, p.colSpan, g.Spacing),
			H: spanExtent(rowH, p.row, p.rowSpan, g.Spacing),
		}
		it.Widget.SetBounds(alignInCell(it.Data.Align, cell, it.Widget.MinSize()))
	}
}

// trackOrigins returns the start coordinate of each track given its size and the
// inter-track spacing, beginning at start.
func trackOrigins(tracks []float64, start, spacing float64) []float64 {
	out := make([]float64, len(tracks))
	x := start
	for i, t := range tracks {
		out[i] = x
		x += t + spacing
	}
	return out
}

// spanExtent sums the sizes of span tracks starting at index, including the
// spacing between them.
func spanExtent(tracks []float64, index, span int, spacing float64) float64 {
	total := 0.0
	for i := 0; i < span; i++ {
		total += tracks[index+i]
	}
	return total + spacing*float64(span-1)
}

// alignInCell positions a child of the given preferred size within a cell. With
// AlignStretch the child fills the cell; otherwise it takes its preferred size
// aligned on both axes.
func alignInCell(a geom.Alignment, cell geom.Rect, pref geom.Size) geom.Rect {
	if a == geom.AlignStretch {
		return cell
	}
	x, w := alignSpan(a, cell.X, cell.W, pref.W)
	y, h := alignSpan(a, cell.Y, cell.H, pref.H)
	return geom.Rect{X: x, Y: y, W: w, H: h}
}
