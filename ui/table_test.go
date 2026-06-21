package ui

import (
	"strconv"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// sortApp builds a table with deliberately unsorted rows; the Age column sorts
// numerically.
func sortApp() (*App, *Table) {
	app := NewApp()
	tbl := NewTable([]Column{
		{Title: "Name"},
		{Title: "Age", Less: func(a, b string) bool { return atoiSafe(a) < atoiSafe(b) }},
	})
	tbl.SetRows([][]string{
		{"Cy", "30"},
		{"Ada", "9"},
		{"Bob", "21"},
	})
	app.SetContent(tbl)
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: tbl.rowHeight() * 5})
	return app, tbl
}

func atoiSafe(s string) int { n, _ := strconv.Atoi(s); return n }

// clickHeader clicks the header cell of column c (200px wide table, equal cols).
func clickHeader(tbl *Table, c int) {
	rh := tbl.rowHeight()
	x := float64(c)*100 + 50 // center of a 100px-wide column
	tbl.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: x, Y: tbl.Bounds().Y + rh/2}})
}

func col0(tbl *Table) []string {
	out := make([]string, len(tbl.rows))
	for i, r := range tbl.rows {
		out[i] = r[0]
	}
	return out
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTableHeaderClickSortsAndToggles(t *testing.T) {
	_, tbl := sortApp()

	clickHeader(tbl, 0) // sort by Name ascending
	if c, asc := tbl.SortColumn(); c != 0 || !asc {
		t.Fatalf("first click should sort col 0 ascending, got (%d,%v)", c, asc)
	}
	if got := col0(tbl); !eqStrings(got, []string{"Ada", "Bob", "Cy"}) {
		t.Fatalf("ascending name order wrong: %v", got)
	}

	clickHeader(tbl, 0) // toggle to descending
	if _, asc := tbl.SortColumn(); asc {
		t.Fatal("second click should flip to descending")
	}
	if got := col0(tbl); !eqStrings(got, []string{"Cy", "Bob", "Ada"}) {
		t.Fatalf("descending name order wrong: %v", got)
	}
}

func TestTableNumericColumnSort(t *testing.T) {
	_, tbl := sortApp()
	clickHeader(tbl, 1) // Age, numeric Less
	if got := col0(tbl); !eqStrings(got, []string{"Ada", "Bob", "Cy"}) {
		t.Fatalf("numeric age sort should order 9,21,30 (Ada,Bob,Cy), got %v", got)
	}
}

func TestTableSortPreservesSelection(t *testing.T) {
	_, tbl := sortApp()
	// Select "Bob" (row index 2 in the unsorted data).
	tbl.SetSelected(2)
	if tbl.rows[tbl.Selected()][0] != "Bob" {
		t.Fatalf("setup: expected Bob selected, got %q", tbl.rows[tbl.Selected()][0])
	}
	clickHeader(tbl, 0) // sort by name → Ada, Bob, Cy
	if tbl.Selected() < 0 || tbl.rows[tbl.Selected()][0] != "Bob" {
		t.Fatalf("selection should follow the row across a sort, got index %d", tbl.Selected())
	}
}

func TestTableNoSortColumn(t *testing.T) {
	app := NewApp()
	clicks := 0
	tbl := NewTable([]Column{{Title: "Name"}, {Title: "Actions", NoSort: true}})
	tbl.OnHeaderClick(func(int) { clicks++ })
	tbl.SetRows([][]string{{"Cy", "x"}, {"Ada", "x"}})
	app.SetContent(tbl)
	tbl.SetBounds(geom.Rect{X: 0, Y: 0, W: 200, H: tbl.rowHeight() * 4})

	before := col0(tbl)
	clickHeader(tbl, 1) // NoSort column
	if c, _ := tbl.SortColumn(); c != -1 {
		t.Fatalf("clicking a NoSort header should not sort, got col %d", c)
	}
	if !eqStrings(col0(tbl), before) {
		t.Fatal("NoSort header click must not reorder rows")
	}
	if clicks != 1 {
		t.Fatalf("OnHeaderClick should still fire for a NoSort column, got %d", clicks)
	}
}

func TestTableSortByProgrammatic(t *testing.T) {
	_, tbl := sortApp()
	tbl.SortBy(0, false) // name descending
	if got := col0(tbl); !eqStrings(got, []string{"Cy", "Bob", "Ada"}) {
		t.Fatalf("SortBy desc wrong: %v", got)
	}
}

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
