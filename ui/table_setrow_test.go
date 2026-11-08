package ui

import (
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

func TestTableSetRowReplacesCells(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetRow(1, "Bea", "Architect")
	if got := tbl.Row(1); !eqStrings(got, []string{"Bea", "Architect"}) {
		t.Fatalf("row 1 = %v", got)
	}
	if tbl.RowCount() != 3 {
		t.Fatalf("row count changed to %d", tbl.RowCount())
	}
}

func TestTableSetRowOutOfRangeIgnored(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetRow(-1, "x", "y")
	tbl.SetRow(3, "x", "y")
	if got := col0(tbl); !eqStrings(got, []string{"Ada", "Bob", "Cy"}) {
		t.Fatalf("rows changed: %v", got)
	}
}

func TestTableSetRowKeepsSelectionAndScroll(t *testing.T) {
	app := NewApp()
	tbl := NewTable([]Column{{Title: "N"}})
	for i := 0; i < 50; i++ {
		tbl.AddRow("r")
	}
	app.SetContent(tbl)
	tbl.SetBounds(geom.Rect{W: 200, H: tbl.rowHeight() * 6})
	tbl.SetSelected(20)
	tbl.offset = tbl.rowHeight() * 10

	tbl.SetRow(20, "changed")
	tbl.SetRow(21, "also")

	if tbl.Selected() != 20 || tbl.Row(20)[0] != "changed" {
		t.Fatalf("selection lost: selected=%d row=%v", tbl.Selected(), tbl.Row(20))
	}
	if tbl.offset != tbl.rowHeight()*10 {
		t.Fatalf("scroll offset reset to %v", tbl.offset)
	}
}

func TestTableSetRowDoesNotResortButNextSortDoes(t *testing.T) {
	_, tbl := tableApp()
	tbl.SortBy(0, true) // Ada, Bob, Cy
	tbl.SetSelected(0)  // Ada

	tbl.SetRow(0, "Zed", "Engineer") // would sort last
	if got := col0(tbl); !eqStrings(got, []string{"Zed", "Bob", "Cy"}) {
		t.Fatalf("SetRow moved rows: %v", got)
	}

	tbl.SortBy(0, true)
	if got := col0(tbl); !eqStrings(got, []string{"Bob", "Cy", "Zed"}) {
		t.Fatalf("sort after SetRow: %v", got)
	}
	// The selection follows the replaced row through the sort.
	if tbl.Selected() != 2 || tbl.Row(2)[0] != "Zed" {
		t.Fatalf("selection did not follow the row: selected=%d", tbl.Selected())
	}
}

func TestTableSetRowMultiSelectionFollowsThroughSort(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetMultiSelect(true)
	tbl.SetSelectedRows(0, 2) // Ada, Cy

	tbl.SetRow(0, "Zed", "Engineer")
	tbl.SortBy(0, true) // Bob, Cy, Zed

	got := []string{}
	for _, i := range tbl.SelectedRows() {
		got = append(got, tbl.Row(i)[0])
	}
	if !eqStrings(got, []string{"Cy", "Zed"}) {
		t.Fatalf("selected after sort: %v", got)
	}
}
