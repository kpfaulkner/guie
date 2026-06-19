package ui

import (
	"testing"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

func tableApp() (*App, *Table) {
	app := NewApp()
	tbl := NewTable([]Column{{Title: "Name"}, {Title: "Role"}})
	tbl.SetRows([][]string{
		{"Ada", "Engineer"},
		{"Bob", "Designer"},
		{"Cy", "Manager"},
	})
	app.SetContent(tbl)
	return app, tbl
}

func TestTableClickSelectsBodyRow(t *testing.T) {
	sel := -1
	app, tbl := tableApp()
	tbl.onSelect = func(i int) { sel = i }
	rh := tbl.rowHeight()
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: rh * 4}) // header + 3 rows
	_ = app

	// Row index 1 lives below the header: y in [bodyTop+rh, bodyTop+2rh).
	y := tbl.Bounds().Y + rh /*header*/ + rh*1 + rh/2
	ev := Event{Type: EventClick, Pos: geom.Point{X: 10, Y: y}}
	tbl.HandleEvent(&ev)

	if tbl.Selected() != 1 || sel != 1 {
		t.Fatalf("clicking body row 1 should select it: selected=%d cb=%d", tbl.Selected(), sel)
	}
}

func TestTableHeaderClickDoesNotSelect(t *testing.T) {
	_, tbl := tableApp()
	rh := tbl.rowHeight()
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: rh * 4})

	// Click within the header row (top rh pixels).
	ev := Event{Type: EventClick, Pos: geom.Point{X: 10, Y: tbl.Bounds().Y + rh/2}}
	tbl.HandleEvent(&ev)
	if tbl.Selected() != -1 {
		t.Fatalf("clicking the header should not select a row, got %d", tbl.Selected())
	}
}

func TestTableKeyboardSelection(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: tbl.rowHeight() * 4})

	keyDown(tbl, render.KeyDown) // -1 → 0
	keyDown(tbl, render.KeyDown) // → 1
	if tbl.Selected() != 1 {
		t.Fatalf("two Downs should select row 1, got %d", tbl.Selected())
	}
	keyDown(tbl, render.KeyEnd)
	if tbl.Selected() != 2 {
		t.Fatalf("End should select the last row, got %d", tbl.Selected())
	}
	keyDown(tbl, render.KeyUp)
	if tbl.Selected() != 1 {
		t.Fatalf("Up should select row 1, got %d", tbl.Selected())
	}
}

func TestTableWheelClamp(t *testing.T) {
	_, tbl := tableApp()
	rh := tbl.rowHeight()
	// Header + 1 visible row; 3 rows of content → scrollable.
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: rh * 2})

	wheel := func(dy float64) {
		ev := Event{Type: EventWheel, Wheel: geom.Point{Y: dy}}
		tbl.HandleEvent(&ev)
	}
	for i := 0; i < 50; i++ {
		wheel(-1)
	}
	if tbl.offset != tbl.maxOffset() || tbl.maxOffset() <= 0 {
		t.Fatalf("should clamp at max offset %v, got %v", tbl.maxOffset(), tbl.offset)
	}
	for i := 0; i < 50; i++ {
		wheel(1)
	}
	if tbl.offset != 0 {
		t.Fatalf("should clamp at 0, got %v", tbl.offset)
	}
}

func TestTableColumnsFillWidth(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: 300}) // tall enough: no overflow, no scrollbar
	widths := tbl.colWidths()
	if len(widths) != 2 {
		t.Fatalf("expected 2 column widths, got %d", len(widths))
	}
	total := widths[0] + widths[1]
	if !approx(total, 200) {
		t.Fatalf("equal columns should fill the width (200), got %v", total)
	}
	if !approx(widths[0], widths[1]) {
		t.Fatalf("default weights should make equal columns, got %v and %v", widths[0], widths[1])
	}
}
