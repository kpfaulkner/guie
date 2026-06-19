package ui

import (
	"math"

	"github.com/kpfaulkner/uiframework/geom"
)

// Grid arranges children into a fixed number of columns, flowing left to right
// and wrapping to a new row every Columns items.
//
// Columns and rows fill the available content rectangle: column widths are
// distributed by ColWeights (equal if unset) and rows share the remaining
// height by RowWeights (equal if unset). Each child is placed within its cell
// per its Align (AlignStretch fills the cell). Cell spanning is not yet
// supported; it is planned for a later iteration.
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

func (g *Grid) rowsFor(n int) int {
	cols := g.cols()
	return (n + cols - 1) / cols
}

// Measure returns the minimum size: each column as wide as its widest child and
// each row as tall as its tallest child, plus inter-cell spacing.
func (g *Grid) Measure(items []Item) geom.Size {
	n := len(items)
	if n == 0 {
		return geom.Size{}
	}
	cols := g.cols()
	rows := g.rowsFor(n)

	colMin := make([]float64, cols)
	rowMin := make([]float64, rows)
	for i, it := range items {
		c, r := i%cols, i/cols
		m := it.Widget.MinSize()
		colMin[c] = math.Max(colMin[c], m.W)
		rowMin[r] = math.Max(rowMin[r], m.H)
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

// Arrange splits the content rectangle into a grid of cells and places each
// child within its cell.
func (g *Grid) Arrange(items []Item, content geom.Rect) {
	n := len(items)
	if n == 0 {
		return
	}
	cols := g.cols()
	rows := g.rowsFor(n)

	colW := distributeTracks(cols, content.W, g.Spacing, g.ColWeights)
	rowH := distributeTracks(rows, content.H, g.Spacing, g.RowWeights)

	// Precompute the top-left of each column and row.
	colX := make([]float64, cols)
	x := content.X
	for c := 0; c < cols; c++ {
		colX[c] = x
		x += colW[c] + g.Spacing
	}
	rowY := make([]float64, rows)
	y := content.Y
	for r := 0; r < rows; r++ {
		rowY[r] = y
		y += rowH[r] + g.Spacing
	}

	for i, it := range items {
		c, r := i%cols, i/cols
		cell := geom.Rect{X: colX[c], Y: rowY[r], W: colW[c], H: rowH[r]}
		it.Widget.SetBounds(alignInCell(it.Data.Align, cell, it.Widget.MinSize()))
	}
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
