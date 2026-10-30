package ui

import (
	"reflect"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	modShift   = render.ModifierSet(render.ModShift)
	modPrimary = render.ModifierSet(render.ModPrimary | render.ModControl)
)

// multiTable returns a table of n rows ("r0".."r<n-1>") with multiple selection
// on, and the callbacks recorded.
type multiRec struct {
	selects []int
	changes [][]int
}

func multiTable(n int) (*Table, *multiRec) {
	tbl := NewTable([]Column{{Title: "Name"}, {Title: "N"}})
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{"r" + string(rune('0'+i)), string(rune('0' + i))}
	}
	tbl.SetRows(rows)
	app := NewApp()
	app.SetContent(tbl)
	tbl.SetBounds(geom.Rect{W: 200, H: tbl.rowHeight() * float64(n+2)})
	tbl.SetMultiSelect(true)
	rec := &multiRec{}
	tbl.OnSelect(func(i int) { rec.selects = append(rec.selects, i) })
	tbl.OnSelectionChange(func(rows []int) { rec.changes = append(rec.changes, append([]int(nil), rows...)) })
	return tbl, rec
}

// clickRowAt clicks row i with the given modifiers.
func clickRowAt(tbl *Table, i int, mods render.ModifierSet) {
	y := tbl.Bounds().Y + tbl.rowHeight()*float64(i+1) + tbl.rowHeight()/2 // +1 skips the header
	tbl.HandleEvent(&Event{Type: EventClick, Pos: geom.Point{X: 10, Y: y}, Button: render.MouseLeft, Modifiers: mods})
}

func key(tbl *Table, k render.Key, mods render.ModifierSet) {
	tbl.HandleEvent(&Event{Type: EventKeyDown, Key: k, Modifiers: mods})
}

func wantRows(t *testing.T, tbl *Table, want ...int) {
	t.Helper()
	got := tbl.SelectedRows()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SelectedRows = %v, want %v", got, want)
	}
}

func TestTableMultiOffByDefaultIsSingleSelect(t *testing.T) {
	tbl, _ := multiTable(5)
	tbl.SetMultiSelect(false)
	var changes [][]int
	tbl.OnSelectionChange(func(r []int) { changes = append(changes, r) })

	clickRowAt(tbl, 1, 0)
	clickRowAt(tbl, 3, modPrimary) // Ctrl-click does nothing special without multi
	clickRowAt(tbl, 0, modShift)
	wantRows(t, tbl, 0)
	if tbl.Selected() != 0 {
		t.Errorf("Selected = %d, want 0", tbl.Selected())
	}
	if !reflect.DeepEqual(changes, [][]int{{1}, {3}, {0}}) {
		t.Errorf("changes = %v", changes)
	}
}

func TestTableMultiPlainClickCollapses(t *testing.T) {
	tbl, rec := multiTable(5)
	clickRowAt(tbl, 1, 0)
	clickRowAt(tbl, 3, modPrimary)
	wantRows(t, tbl, 1, 3)
	clickRowAt(tbl, 2, 0)
	wantRows(t, tbl, 2)
	if got := rec.changes[len(rec.changes)-1]; !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("last change = %v", got)
	}
}

func TestTableMultiCtrlClickToggles(t *testing.T) {
	tbl, rec := multiTable(5)
	clickRowAt(tbl, 1, modPrimary)
	clickRowAt(tbl, 3, modPrimary)
	wantRows(t, tbl, 1, 3)
	clickRowAt(tbl, 1, modPrimary) // off again
	wantRows(t, tbl, 3)
	clickRowAt(tbl, 3, modPrimary) // off: nothing left
	wantRows(t, tbl)
	want := [][]int{{1}, {1, 3}, {3}, nil}
	if !reflect.DeepEqual(rec.changes, want) {
		t.Errorf("changes = %v, want %v", rec.changes, want)
	}
	if tbl.Selected() != 3 {
		t.Errorf("active row = %d, want 3 (the last one clicked)", tbl.Selected())
	}
}

func TestTableMultiShiftClickRange(t *testing.T) {
	tbl, _ := multiTable(7)
	clickRowAt(tbl, 2, 0)
	clickRowAt(tbl, 5, modShift)
	wantRows(t, tbl, 2, 3, 4, 5)
	clickRowAt(tbl, 0, modShift) // the anchor stays at 2: the range flips
	wantRows(t, tbl, 0, 1, 2)
	clickRowAt(tbl, 3, 0)
	clickRowAt(tbl, 6, modShift)
	wantRows(t, tbl, 3, 4, 5, 6)
}

func TestTableMultiCtrlShiftClickAddsRange(t *testing.T) {
	tbl, _ := multiTable(8)
	clickRowAt(tbl, 0, 0)
	clickRowAt(tbl, 4, modPrimary) // 0 and 4, anchor 4
	clickRowAt(tbl, 6, modShift|modPrimary)
	wantRows(t, tbl, 0, 4, 5, 6)
}

func TestTableMultiShiftClickWithNoAnchorSelectsOnlyThatRow(t *testing.T) {
	tbl, _ := multiTable(5)
	clickRowAt(tbl, 3, modShift)
	wantRows(t, tbl, 3)
}

func TestTableMultiKeyboard(t *testing.T) {
	tbl, rec := multiTable(6)
	key(tbl, render.KeyDown, 0) // selects row 0
	wantRows(t, tbl, 0)
	key(tbl, render.KeyDown, modShift)
	key(tbl, render.KeyDown, modShift)
	wantRows(t, tbl, 0, 1, 2)
	key(tbl, render.KeyUp, modShift) // shrinks back towards the anchor
	wantRows(t, tbl, 0, 1)
	key(tbl, render.KeyDown, 0) // plain move collapses
	wantRows(t, tbl, 2)
	key(tbl, render.KeyEnd, modShift)
	wantRows(t, tbl, 2, 3, 4, 5)
	key(tbl, render.KeyHome, modShift)
	wantRows(t, tbl, 0, 1, 2)
	if len(rec.selects) == 0 || rec.selects[0] != 0 {
		t.Errorf("OnSelect should report the active row; got %v", rec.selects)
	}
}

func TestTableMultiShiftDownFromNothingStartsAtTop(t *testing.T) {
	tbl, _ := multiTable(4)
	key(tbl, render.KeyDown, modShift)
	wantRows(t, tbl, 0)
	key(tbl, render.KeyDown, modShift)
	wantRows(t, tbl, 0, 1)
}

func TestTableMultiSelectAll(t *testing.T) {
	tbl, _ := multiTable(4)
	clickRowAt(tbl, 2, 0)
	key(tbl, render.KeyA, modPrimary)
	wantRows(t, tbl, 0, 1, 2, 3)
	if tbl.Selected() != 2 {
		t.Errorf("select all must keep the active row; got %d", tbl.Selected())
	}
	key(tbl, render.KeyA, 0) // not a shortcut without the modifier
	wantRows(t, tbl, 0, 1, 2, 3)
}

func TestTableSelectAllIgnoredWithoutMulti(t *testing.T) {
	tbl, _ := multiTable(4)
	tbl.SetMultiSelect(false)
	clickRowAt(tbl, 1, 0)
	if key(tbl, render.KeyA, modPrimary); len(tbl.SelectedRows()) != 1 {
		t.Errorf("Ctrl+A must not select all without multi; got %v", tbl.SelectedRows())
	}
}

func TestTableMultiSetSelectedRows(t *testing.T) {
	tbl, rec := multiTable(6)
	tbl.SetSelectedRows(4, 1, 99, -2, 3)
	wantRows(t, tbl, 1, 3, 4)
	if tbl.Selected() != 3 {
		t.Errorf("last valid row should be active, got %d", tbl.Selected())
	}
	if len(rec.changes) != 1 || !reflect.DeepEqual(rec.changes[0], []int{1, 3, 4}) {
		t.Errorf("changes = %v", rec.changes)
	}
	tbl.SetSelectedRows(1, 3, 4) // same set again: no change reported
	if len(rec.changes) != 1 {
		t.Errorf("an unchanged selection must not be reported again: %v", rec.changes)
	}
	tbl.SetSelectedRows()
	wantRows(t, tbl)
	if tbl.Selected() != -1 {
		t.Errorf("Selected = %d, want -1", tbl.Selected())
	}
}

func TestTableSetSelectedRowsWithoutMultiSelectsLast(t *testing.T) {
	tbl, _ := multiTable(5)
	tbl.SetMultiSelect(false)
	tbl.SetSelectedRows(1, 3)
	wantRows(t, tbl, 3)
	tbl.SetSelectedRows()
	wantRows(t, tbl)
}

func TestTableMultiSetSelectedCollapses(t *testing.T) {
	tbl, _ := multiTable(5)
	tbl.SetSelectedRows(0, 1, 2)
	tbl.SetSelected(2) // the active row is already 2, but the set is bigger
	wantRows(t, tbl, 2)
}

func TestTableMultiSetRowsClearsSelection(t *testing.T) {
	tbl, _ := multiTable(5)
	tbl.SetSelectedRows(1, 2)
	tbl.SetRows([][]string{{"x", "1"}, {"y", "2"}})
	wantRows(t, tbl)
	// And the first Shift-click after is not tied to the old anchor.
	clickRowAt(tbl, 1, modShift)
	wantRows(t, tbl, 1)
}

func TestTableMultiSelectionFollowsRowsThroughSort(t *testing.T) {
	tbl := NewTable([]Column{{Title: "Name"}})
	tbl.SetRows([][]string{{"d"}, {"b"}, {"c"}, {"a"}})
	app := NewApp()
	app.SetContent(tbl)
	tbl.SetBounds(geom.Rect{W: 200, H: tbl.rowHeight() * 8})
	tbl.SetMultiSelect(true)

	tbl.SetSelectedRows(0, 2) // d and c
	tbl.SortBy(0, true)       // a b c d
	got := []string{}
	for _, i := range tbl.SelectedRows() {
		got = append(got, tbl.Row(i)[0])
	}
	if !reflect.DeepEqual(got, []string{"c", "d"}) {
		t.Errorf("selected after sort = %v, want [c d]", got)
	}
	if tbl.Selected() < 0 || tbl.Row(tbl.Selected())[0] != "c" {
		t.Errorf("active row should still be c, got %d", tbl.Selected())
	}
	// The anchor followed too: shift-click from the active row's anchor.
	clickRowAt(tbl, 0, modShift) // anchor is c (row 2) -> rows 0..2
	if !reflect.DeepEqual(tbl.SelectedRows(), []int{0, 1, 2}) {
		t.Errorf("range from the moved anchor = %v", tbl.SelectedRows())
	}
}

func TestTableSetMultiSelectKeepsActiveRowThenCollapses(t *testing.T) {
	tbl := NewTable([]Column{{Title: "Name"}})
	tbl.SetRows([][]string{{"a"}, {"b"}, {"c"}})
	tbl.SetSelected(1)
	tbl.SetMultiSelect(true)
	wantRows(t, tbl, 1)
	tbl.SetSelectedRows(0, 1, 2)
	tbl.SetMultiSelect(false)
	wantRows(t, tbl, tbl.Selected())
	if tbl.MultiSelect() {
		t.Error("MultiSelect should be off")
	}
}

func TestTableMultiDrawsEverySelectedRowHighlighted(t *testing.T) {
	tbl, _ := multiTable(4)
	tbl.SetSelectedRows(0, 2)
	for i, want := range []bool{true, false, true, false} {
		if tbl.isSelected(i) != want {
			t.Errorf("isSelected(%d) = %v, want %v", i, tbl.isSelected(i), want)
		}
	}
}
