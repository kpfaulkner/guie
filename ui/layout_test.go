package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

// stub is a leaf widget with a fixed minimum size, used to drive layout tests.
type stub struct {
	BaseWidget
	min geom.Size
}

func newStub(w, h float64) *stub {
	return &stub{BaseWidget: NewBase(), min: geom.Size{W: w, H: h}}
}

func (s *stub) MinSize() geom.Size { return s.min }

func items(ws ...Widget) []Item {
	its := make([]Item, len(ws))
	for i, w := range ws {
		its[i] = Item{Widget: w, Data: defaultLayoutData()}
	}
	return its
}

func approx(a, b float64) bool {
	d := a - b
	return d < 0.001 && d > -0.001
}

func wantRect(t *testing.T, name string, got, exp geom.Rect) {
	t.Helper()
	if !approx(got.X, exp.X) || !approx(got.Y, exp.Y) || !approx(got.W, exp.W) || !approx(got.H, exp.H) {
		t.Errorf("%s: got %+v, want %+v", name, got, exp)
	}
}

func TestBoxHorizontalWeights(t *testing.T) {
	a, b := newStub(10, 5), newStub(10, 5)
	its := items(a, b)
	its[0].Data.Weight = 1
	its[1].Data.Weight = 3

	box := HBox(0)
	box.Arrange(its, geom.Rect{X: 0, Y: 0, W: 100, H: 20})

	// free = 100 - (10+10) = 80; a gets 1/4 (20)→30, b gets 3/4 (60)→70.
	wantRect(t, "a", a.Bounds(), geom.Rect{X: 0, Y: 0, W: 30, H: 20})
	wantRect(t, "b", b.Bounds(), geom.Rect{X: 30, Y: 0, W: 70, H: 20})
}

func TestBoxHorizontalSpacing(t *testing.T) {
	a, b := newStub(10, 5), newStub(10, 5)
	its := items(a, b)
	its[0].Data.Weight = 1
	its[1].Data.Weight = 1

	box := HBox(10)
	box.Arrange(its, geom.Rect{X: 0, Y: 0, W: 100, H: 20})

	// free = 100 - 20 - 10(spacing) = 70; split evenly → each 10+35 = 45.
	wantRect(t, "a", a.Bounds(), geom.Rect{X: 0, Y: 0, W: 45, H: 20})
	wantRect(t, "b", b.Bounds(), geom.Rect{X: 55, Y: 0, W: 45, H: 20})
}

func TestBoxVerticalCrossAlign(t *testing.T) {
	c := newStub(40, 10)
	its := items(c)
	its[0].Data.Align = geom.AlignCenter

	VBox(0).Arrange(its, geom.Rect{X: 0, Y: 0, W: 100, H: 10})

	// Cross axis (horizontal) centers the 40-wide child in 100: x = 30.
	wantRect(t, "c", c.Bounds(), geom.Rect{X: 30, Y: 0, W: 40, H: 10})
}

func TestBoxNoWeightStaysAtMin(t *testing.T) {
	a, b := newStub(10, 5), newStub(10, 5)
	HBox(0).Arrange(items(a, b), geom.Rect{X: 0, Y: 0, W: 100, H: 20})

	// No weights: children keep min width and sit at the start.
	wantRect(t, "a", a.Bounds(), geom.Rect{X: 0, Y: 0, W: 10, H: 20})
	wantRect(t, "b", b.Bounds(), geom.Rect{X: 10, Y: 0, W: 10, H: 20})
}

func TestStackCenter(t *testing.T) {
	c := newStub(20, 10)
	its := items(c)
	its[0].Data.Align = geom.AlignCenter

	NewStack().Arrange(its, geom.Rect{X: 0, Y: 0, W: 100, H: 50})

	wantRect(t, "c", c.Bounds(), geom.Rect{X: 40, Y: 20, W: 20, H: 10})
}

func TestStackStretchFills(t *testing.T) {
	c := newStub(20, 10)
	NewStack().Arrange(items(c), geom.Rect{X: 5, Y: 5, W: 90, H: 40})

	wantRect(t, "c", c.Bounds(), geom.Rect{X: 5, Y: 5, W: 90, H: 40})
}

func TestGridEqualCells(t *testing.T) {
	cells := []Widget{newStub(1, 1), newStub(1, 1), newStub(1, 1), newStub(1, 1)}
	// 2 columns, 4 items → 2 rows. Spacing 0, content 100x100 → 50x50 cells.
	NewGrid(2, 0).Arrange(items(cells...), geom.Rect{X: 0, Y: 0, W: 100, H: 100})

	wantRect(t, "cell0", cells[0].Bounds(), geom.Rect{X: 0, Y: 0, W: 50, H: 50})
	wantRect(t, "cell1", cells[1].Bounds(), geom.Rect{X: 50, Y: 0, W: 50, H: 50})
	wantRect(t, "cell2", cells[2].Bounds(), geom.Rect{X: 0, Y: 50, W: 50, H: 50})
	wantRect(t, "cell3", cells[3].Bounds(), geom.Rect{X: 50, Y: 50, W: 50, H: 50})
}

func TestGridSpacing(t *testing.T) {
	cells := []Widget{newStub(1, 1), newStub(1, 1)}
	// 2 columns, spacing 10, width 110 → cells of 50 each with a 10 gap.
	NewGrid(2, 10).Arrange(items(cells...), geom.Rect{X: 0, Y: 0, W: 110, H: 20})

	wantRect(t, "cell0", cells[0].Bounds(), geom.Rect{X: 0, Y: 0, W: 50, H: 20})
	wantRect(t, "cell1", cells[1].Bounds(), geom.Rect{X: 60, Y: 0, W: 50, H: 20})
}

func TestBoxMeasure(t *testing.T) {
	got := HBox(10).Measure(items(newStub(10, 5), newStub(20, 8)))
	// main = 10+20+10(spacing) = 40; cross = max(5,8) = 8.
	if !approx(got.W, 40) || !approx(got.H, 8) {
		t.Errorf("Measure: got %+v, want {40 8}", got)
	}
}
