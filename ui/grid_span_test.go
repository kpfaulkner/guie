package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

func spanItem(w Widget, cols, rows int) Item {
	return Item{Widget: w, Data: LayoutData{Align: geom.AlignStretch, ColSpan: cols, RowSpan: rows}}
}

func cellItem(w Widget) Item {
	return Item{Widget: w, Data: defaultLayoutData()}
}

func TestGridColSpan(t *testing.T) {
	header := newStub(1, 1)
	a, b, c := newStub(1, 1), newStub(1, 1), newStub(1, 1)
	its := []Item{spanItem(header, 3, 1), cellItem(a), cellItem(b), cellItem(c)}

	// 3 columns, no spacing, 120x100 → columns 40 each, 2 rows of 50.
	NewGrid(3, 0).Arrange(its, geom.Rect{X: 0, Y: 0, W: 120, H: 100})

	wantRect(t, "header", header.Bounds(), geom.Rect{X: 0, Y: 0, W: 120, H: 50})
	wantRect(t, "a", a.Bounds(), geom.Rect{X: 0, Y: 50, W: 40, H: 50})
	wantRect(t, "b", b.Bounds(), geom.Rect{X: 40, Y: 50, W: 40, H: 50})
	wantRect(t, "c", c.Bounds(), geom.Rect{X: 80, Y: 50, W: 40, H: 50})
}

func TestGridRowSpan(t *testing.T) {
	side := newStub(1, 1)
	a, b, c := newStub(1, 1), newStub(1, 1), newStub(1, 1)
	// side spans 2 rows in column 0; the rest auto-flow around it.
	its := []Item{spanItem(side, 1, 2), cellItem(a), cellItem(b), cellItem(c)}

	// 2 columns, no spacing, 100x150 → columns 50, rows 50 each (3 rows).
	NewGrid(2, 0).Arrange(its, geom.Rect{X: 0, Y: 0, W: 100, H: 150})

	wantRect(t, "side", side.Bounds(), geom.Rect{X: 0, Y: 0, W: 50, H: 100}) // spans rows 0-1
	wantRect(t, "a", a.Bounds(), geom.Rect{X: 50, Y: 0, W: 50, H: 50})
	wantRect(t, "b", b.Bounds(), geom.Rect{X: 50, Y: 50, W: 50, H: 50})
	wantRect(t, "c", c.Bounds(), geom.Rect{X: 0, Y: 100, W: 50, H: 50}) // wraps below the sidebar
}

func TestGridSpanSpacing(t *testing.T) {
	header := newStub(1, 1)
	a, b := newStub(1, 1), newStub(1, 1)
	its := []Item{spanItem(header, 2, 1), cellItem(a), cellItem(b)}

	// 2 columns, spacing 10, width 110 → each column 50, gap 10.
	NewGrid(2, 10).Arrange(its, geom.Rect{X: 0, Y: 0, W: 110, H: 110})

	// Header spans both columns AND the gap between them: 50 + 10 + 50 = 110.
	if got := header.Bounds().W; !approx(got, 110) {
		t.Fatalf("spanning header width should include the inter-column gap: got %v want 110", got)
	}
}

func TestGridSpanClampedToColumns(t *testing.T) {
	wide := newStub(1, 1)
	// Asking for 5 columns in a 2-column grid clamps to 2.
	NewGrid(2, 0).Arrange([]Item{spanItem(wide, 5, 1)}, geom.Rect{X: 0, Y: 0, W: 100, H: 40})
	if got := wide.Bounds().W; !approx(got, 100) {
		t.Fatalf("over-wide span should clamp to the full width (100), got %v", got)
	}
}
