package ui

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

// longTable returns a laid-out table of n rows "0".."n-1", tall enough to show
// visible rows at a time.
func longTable(n, visible int) *Table {
	app := NewApp()
	tbl := NewTable([]Column{{Title: "N"}})
	for i := 0; i < n; i++ {
		tbl.AddRow(strconv.Itoa(i))
	}
	app.SetContent(tbl)
	tbl.SetBounds(geom.Rect{W: 200, H: tbl.rowHeight() * float64(visible+1)}) // +1 for the header
	return tbl
}

func TestTableRemoveRowShiftsRowsAndSelection(t *testing.T) {
	_, tbl := tableApp() // Ada, Bob, Cy
	tbl.SetSelected(2)   // Cy
	var changes [][]int
	tbl.OnSelectionChange(func(r []int) { changes = append(changes, r) })

	tbl.RemoveRow(0)
	if got := col0(tbl); !eqStrings(got, []string{"Bob", "Cy"}) {
		t.Fatalf("rows %v", got)
	}
	if tbl.Selected() != 1 || tbl.Row(1)[0] != "Cy" {
		t.Fatalf("selection did not follow Cy: %d", tbl.Selected())
	}
	if !reflect.DeepEqual(changes, [][]int{{1}}) {
		t.Fatalf("changes %v", changes)
	}
}

func TestTableRemoveSelectedRowDeselects(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetSelected(1) // Bob
	var changes [][]int
	selects := 0
	tbl.OnSelectionChange(func(r []int) { changes = append(changes, r) })
	tbl.OnSelect(func(int) { selects++ })

	tbl.RemoveRow(1)
	if tbl.Selected() != -1 || len(tbl.SelectedRows()) != 0 {
		t.Fatalf("still selected: %d", tbl.Selected())
	}
	if len(changes) != 1 || len(changes[0]) != 0 || selects != 0 {
		t.Fatalf("changes %v selects %d", changes, selects)
	}
}

func TestTableRemoveRowMultiSelection(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetMultiSelect(true)
	tbl.SetSelectedRows(0, 2) // Ada, Cy
	tbl.RemoveRow(0)
	if got := tbl.SelectedRows(); !reflect.DeepEqual(got, []int{1}) || tbl.Row(1)[0] != "Cy" {
		t.Fatalf("selected %v", got)
	}
}

func TestTableRemoveRowOutOfRangeIgnored(t *testing.T) {
	_, tbl := tableApp()
	tbl.RemoveRow(-1)
	tbl.RemoveRow(3)
	if tbl.RowCount() != 3 {
		t.Fatalf("rows %d", tbl.RowCount())
	}
}

func TestTableRemoveRowAboveViewKeepsContentStill(t *testing.T) {
	tbl := longTable(50, 5)
	rh := tbl.rowHeight()
	tbl.offset = rh * 20 // row 20 at the top
	tbl.RemoveRow(3)     // above the view
	if tbl.offset != rh*19 || tbl.Row(19)[0] != "20" {
		t.Fatalf("offset %v (want %v), top row %v", tbl.offset, rh*19, tbl.Row(19))
	}
	tbl.RemoveRow(30) // below the top: no change
	if tbl.offset != rh*19 {
		t.Fatalf("removing below the view moved it: %v", tbl.offset)
	}
}

func TestTableScrolledToBottomAndScrollToRow(t *testing.T) {
	tbl := longTable(50, 5)
	if tbl.ScrolledToBottom() {
		t.Fatal("at the top of 50 rows, reported at bottom")
	}
	tbl.ScrollToRow(49)
	if !tbl.ScrolledToBottom() || tbl.offset != tbl.maxOffset() {
		t.Fatalf("ScrollToRow(last) did not reach the bottom: %v of %v", tbl.offset, tbl.maxOffset())
	}
	tbl.ScrollToRow(10)
	if tbl.offset != tbl.rowHeight()*10 || tbl.ScrolledToBottom() {
		t.Fatalf("ScrollToRow(10): offset %v", tbl.offset)
	}
	tbl.ScrollToRow(99) // ignored
	if tbl.offset != tbl.rowHeight()*10 {
		t.Fatal("out-of-range ScrollToRow moved the view")
	}
}

func TestTableScrolledToBottomWhenEverythingFitsOrNotLaidOut(t *testing.T) {
	if !longTable(3, 5).ScrolledToBottom() {
		t.Fatal("short table not at bottom")
	}
	tbl := NewTable([]Column{{Title: "N"}})
	tbl.AddRow("x")
	if !tbl.ScrolledToBottom() {
		t.Fatal("unlaid-out table not at bottom")
	}
	tbl.ScrollToRow(0) // no size yet: must not panic or move
}

func TestTableFollowPattern(t *testing.T) {
	tbl := longTable(5, 5)
	for i := 5; i < 20; i++ {
		follow := tbl.ScrolledToBottom()
		tbl.AddRow(strconv.Itoa(i))
		if follow {
			tbl.ScrollToRow(tbl.RowCount() - 1)
		}
	}
	if !tbl.ScrolledToBottom() {
		t.Fatal("following lost the bottom")
	}
}

func TestTableClearSelection(t *testing.T) {
	_, tbl := tableApp()
	tbl.SetSelected(1)
	var changes [][]int
	tbl.OnSelectionChange(func(r []int) { changes = append(changes, r) })
	tbl.ClearSelection()
	tbl.ClearSelection() // nothing selected: not reported again
	if tbl.Selected() != -1 || len(changes) != 1 {
		t.Fatalf("selected %d, changes %v", tbl.Selected(), changes)
	}
}
