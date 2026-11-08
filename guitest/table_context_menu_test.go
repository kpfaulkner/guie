package guitest_test

import (
	"strconv"
	"testing"

	"github.com/kpfaulkner/guie/guitest"
	"github.com/kpfaulkner/guie/ui"
)

// menuTable is a window filled by a table of n rows, row i saying "row i",
// that records which rows it is asked for a menu for.
func menuTable(n int) (*guitest.Harness, *ui.Table, *[]int) {
	h := guitest.New(400, 200)
	tbl := ui.NewTable([]ui.Column{{Title: "Name", Weight: 1}})
	for i := range n {
		tbl.AddRow("row " + strconv.Itoa(i))
	}
	asked := &[]int{}
	tbl.OnRowContextMenu(func(row int) []ui.MenuItem {
		*asked = append(*asked, row)
		return []ui.MenuItem{ui.NewMenuItem("For "+tbl.Row(row)[0], func() {})}
	})
	h.SetContent(tbl)
	h.MoveMouse(-1, -1)
	h.Step()
	return h, tbl, asked
}

func TestRowContextMenuIsForTheRowUnderThePointer(t *testing.T) {
	h, _, asked := menuTable(3)
	op := textOp(t, h.Frame(), "row 1")
	r := h.RightClick(op.A.X+2, op.A.Y+2)
	if !r.HasText("For row 1") {
		t.Fatalf("texts %v", r.Texts())
	}
	if len(*asked) != 1 || (*asked)[0] != 1 {
		t.Fatalf("asked for %v, want row 1 once", *asked)
	}
}

func TestRowContextMenuFollowsRowsScrollingUnderAStillPointer(t *testing.T) {
	h, tbl, asked := menuTable(50)
	first := textOp(t, h.Frame(), "row 0")
	x, y := first.A.X+2, first.A.Y+2

	// The pointer goes to the first row and stays there.
	h.MoveMouse(x, y)
	h.Step()
	tbl.ScrollToRow(49)
	r := h.Step()
	var under string
	for _, op := range r.OpsOfKind(guitest.OpDrawText) {
		if op.A.Y <= y && y < op.A.Y+16 && op.Text != "Name" {
			under = op.Text
		}
	}
	if under == "" || under == "row 0" {
		t.Fatalf("the rows did not scroll; under the pointer is %q", under)
	}
	want, _ := strconv.Atoi(under[len("row "):])

	r = h.RightClick(x, y)
	if len(*asked) != 1 || (*asked)[0] != want || !r.HasText("For "+under) {
		t.Fatalf("asked for %v, want %d; texts %v", *asked, want, r.Texts())
	}
}

func TestRowContextMenuIsForTheDisplayedRowWhenSorted(t *testing.T) {
	h, tbl, asked := menuTable(3)
	tbl.SortBy(0, false) // row 2, row 1, row 0
	r := h.Step()
	op := textOp(t, r, "row 2")
	r = h.RightClick(op.A.X+2, op.A.Y+2)
	if len(*asked) != 1 || (*asked)[0] != 0 || !r.HasText("For row 2") {
		t.Fatalf("asked for %v; texts %v", *asked, r.Texts())
	}
}

func TestNoRowContextMenuOffTheRows(t *testing.T) {
	h, _, asked := menuTable(3)
	head := textOp(t, h.Frame(), "Name")
	h.RightClick(head.A.X+2, head.A.Y+2)
	last := textOp(t, h.Frame(), "row 2")
	h.RightClick(last.A.X+2, last.A.Y+60) // below the last row
	if len(*asked) != 0 {
		t.Fatalf("asked for %v", *asked)
	}
}

func TestSetContextMenuIsUsedOffTheRows(t *testing.T) {
	h, tbl, asked := menuTable(3)
	tbl.SetContextMenu(ui.NewMenuItem("Whole table", func() {}))
	head := textOp(t, h.Frame(), "Name")
	if r := h.RightClick(head.A.X+2, head.A.Y+2); !r.HasText("Whole table") || len(*asked) != 0 {
		t.Fatalf("asked for %v; texts %v", *asked, r.Texts())
	}
}

func TestTableWithoutARowMenuStillHasItsContextMenu(t *testing.T) {
	h := guitest.New(400, 200)
	tbl := ui.NewTable([]ui.Column{{Title: "Name", Weight: 1}})
	tbl.AddRow("a row")
	tbl.SetContextMenu(ui.NewMenuItem("Whole table", func() {}))
	h.SetContent(tbl)
	h.Step()
	op := textOp(t, h.Frame(), "a row")
	if r := h.RightClick(op.A.X+2, op.A.Y+2); !r.HasText("Whole table") {
		t.Fatalf("texts %v", r.Texts())
	}
}

func TestRowContextMenuReturningNothingShowsNoMenu(t *testing.T) {
	h, tbl, _ := menuTable(3)
	tbl.OnRowContextMenu(func(int) []ui.MenuItem { return nil })
	op := textOp(t, h.Frame(), "row 1")
	before := len(h.Step().Texts())
	if r := h.RightClick(op.A.X+2, op.A.Y+2); len(r.Texts()) != before {
		t.Fatalf("texts %v", r.Texts())
	}
}

func TestRowContextMenuNotOpenedFromTheScrollbar(t *testing.T) {
	h, tbl, asked := menuTable(50)
	b := tbl.Bounds()
	h.RightClick(b.X+b.W-3, b.Y+b.H/2)
	if len(*asked) != 0 {
		t.Fatalf("asked for %v", *asked)
	}
}
