package ui

import (
	"reflect"
	"testing"

	"github.com/kpfaulkner/guie/render"
)

const modPrimaryOnly = render.ModifierSet(render.ModPrimary | render.ModControl)

func TestTableCtrlArrowMovesCursorWithoutChangingSelection(t *testing.T) {
	tbl, rec := multiTable(6)
	clickRowAt(tbl, 1, 0)
	selects, changes := len(rec.selects), len(rec.changes)

	key(tbl, render.KeyDown, modPrimaryOnly)
	key(tbl, render.KeyDown, modPrimaryOnly)
	if tbl.Selected() != 3 {
		t.Errorf("cursor = %d, want 3", tbl.Selected())
	}
	wantRows(t, tbl, 1)
	if len(rec.selects) != selects || len(rec.changes) != changes {
		t.Errorf("moving the cursor must report nothing: selects %v, changes %v", rec.selects[selects:], rec.changes[changes:])
	}

	key(tbl, render.KeyUp, modPrimaryOnly)
	if tbl.Selected() != 2 {
		t.Errorf("cursor = %d, want 2", tbl.Selected())
	}
	key(tbl, render.KeyHome, modPrimaryOnly)
	if tbl.Selected() != 0 {
		t.Errorf("Ctrl+Home cursor = %d, want 0", tbl.Selected())
	}
	key(tbl, render.KeyEnd, modPrimaryOnly)
	if tbl.Selected() != 5 {
		t.Errorf("Ctrl+End cursor = %d, want 5", tbl.Selected())
	}
	wantRows(t, tbl, 1)

	// Off either end nothing happens.
	key(tbl, render.KeyDown, modPrimaryOnly)
	if tbl.Selected() != 5 {
		t.Errorf("Ctrl+Down past the end moved the cursor to %d", tbl.Selected())
	}
	tbl.moveCursor(0)
	key(tbl, render.KeyUp, modPrimaryOnly)
	if tbl.Selected() != 0 {
		t.Errorf("Ctrl+Up past the start moved the cursor to %d", tbl.Selected())
	}
}

func TestTableSpaceTogglesTheCursorRow(t *testing.T) {
	tbl, rec := multiTable(6)
	clickRowAt(tbl, 1, 0)
	key(tbl, render.KeyDown, modPrimaryOnly) // cursor on 2
	key(tbl, render.KeySpace, 0)
	wantRows(t, tbl, 1, 2)
	key(tbl, render.KeySpace, 0)
	wantRows(t, tbl, 1)
	if tbl.Selected() != 2 {
		t.Errorf("the cursor stays on the row Space acted on; got %d", tbl.Selected())
	}
	key(tbl, render.KeySpace, modPrimaryOnly) // Ctrl+Space does the same
	wantRows(t, tbl, 1, 2)

	want := [][]int{{1}, {1, 2}, {1}, {1, 2}}
	if !reflect.DeepEqual(rec.changes, want) {
		t.Errorf("changes = %v, want %v", rec.changes, want)
	}
}

func TestTableSpaceWithNoCursorSelectsTheFirstRow(t *testing.T) {
	tbl, _ := multiTable(4)
	key(tbl, render.KeySpace, 0)
	wantRows(t, tbl, 0)
	if tbl.Selected() != 0 {
		t.Errorf("cursor = %d, want 0", tbl.Selected())
	}
}

func TestTableCtrlArrowFromNothingStartsAtTheTop(t *testing.T) {
	tbl, _ := multiTable(4)
	key(tbl, render.KeyDown, modPrimaryOnly)
	if tbl.Selected() != 0 {
		t.Errorf("cursor = %d, want 0", tbl.Selected())
	}
	wantRows(t, tbl)
}

func TestTableKeyboardOnlyNonContiguousSelection(t *testing.T) {
	tbl, _ := multiTable(8)
	key(tbl, render.KeyDown, 0) // selects row 0
	key(tbl, render.KeyDown, modPrimaryOnly)
	key(tbl, render.KeyDown, modPrimaryOnly) // cursor 2
	key(tbl, render.KeySpace, 0)             // {0, 2}, anchor 2
	wantRows(t, tbl, 0, 2)

	key(tbl, render.KeyDown, modPrimaryOnly)
	key(tbl, render.KeyDown, modPrimaryOnly) // cursor 4
	key(tbl, render.KeySpace, 0)             // {0, 2, 4}, anchor 4
	wantRows(t, tbl, 0, 2, 4)

	// Ctrl+Shift extends from the anchor, adding to what is already selected.
	key(tbl, render.KeyDown, modPrimaryOnly|modShift)
	key(tbl, render.KeyDown, modPrimaryOnly|modShift)
	wantRows(t, tbl, 0, 2, 4, 5, 6)
	if tbl.Selected() != 6 {
		t.Errorf("cursor = %d, want 6", tbl.Selected())
	}

	// A plain arrow collapses it all again.
	key(tbl, render.KeyUp, 0)
	wantRows(t, tbl, 5)
}

func TestTableCtrlShiftHomeEndAddRanges(t *testing.T) {
	tbl, _ := multiTable(6)
	clickRowAt(tbl, 3, 0)
	key(tbl, render.KeyEnd, modPrimaryOnly|modShift)
	wantRows(t, tbl, 3, 4, 5)
	key(tbl, render.KeyHome, modPrimaryOnly|modShift)
	wantRows(t, tbl, 0, 1, 2, 3, 4, 5)
}

func TestTableSpaceAndCtrlArrowDoNothingSpecialWithoutMulti(t *testing.T) {
	tbl, _ := multiTable(5)
	tbl.SetMultiSelect(false)
	if handled := tbl.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeySpace}); handled {
		t.Error("Space must be left to others when multiple selection is off")
	}
	clickRowAt(tbl, 1, 0)
	key(tbl, render.KeyDown, modPrimaryOnly) // plain Down, as before
	wantRows(t, tbl, 2)
}

func TestTableAltSpaceIsNotHandled(t *testing.T) {
	tbl, _ := multiTable(3)
	if handled := tbl.HandleEvent(&Event{Type: EventKeyDown, Key: render.KeySpace, Modifiers: render.ModifierSet(render.ModAlt)}); handled {
		t.Error("Alt+Space belongs to the window system")
	}
}

func TestTableOnSelectIsOnlyForRowsThatBecomeSelected(t *testing.T) {
	tbl, rec := multiTable(6)
	clickRowAt(tbl, 2, 0)
	clickRowAt(tbl, 4, modPrimaryOnly) // on: reported
	if !reflect.DeepEqual(rec.selects, []int{2, 4}) {
		t.Fatalf("selects = %v, want [2 4]", rec.selects)
	}

	// Deselecting a row other than the active one moves the cursor onto it, but
	// nothing became selected, so nothing is reported.
	clickRowAt(tbl, 2, modPrimaryOnly)
	wantRows(t, tbl, 4)
	if tbl.Selected() != 2 {
		t.Errorf("the cursor should be on the row just clicked, got %d", tbl.Selected())
	}
	if !reflect.DeepEqual(rec.selects, []int{2, 4}) {
		t.Errorf("OnSelect fired for a row that was deselected: %v", rec.selects)
	}

	// Toggling it back on is reported, though the cursor was already there.
	clickRowAt(tbl, 2, modPrimaryOnly)
	if !reflect.DeepEqual(rec.selects, []int{2, 4, 2}) {
		t.Errorf("selects = %v, want [2 4 2]", rec.selects)
	}
	// So is Space doing the same, and Space deselecting is not.
	key(tbl, render.KeySpace, 0) // row 2 off
	key(tbl, render.KeySpace, 0) // row 2 on
	if !reflect.DeepEqual(rec.selects, []int{2, 4, 2, 2}) {
		t.Errorf("selects = %v, want [2 4 2 2]", rec.selects)
	}
	// Select all does not re-report a row that was already selected.
	key(tbl, render.KeyA, modPrimaryOnly)
	if !reflect.DeepEqual(rec.selects, []int{2, 4, 2, 2}) {
		t.Errorf("select-all reported the active row again: %v", rec.selects)
	}
}

func TestTableCursorRowSurvivesASort(t *testing.T) {
	tbl := NewTable([]Column{{Title: "Name"}})
	tbl.SetRows([][]string{{"d"}, {"b"}, {"c"}, {"a"}})
	tbl.SetMultiSelect(true)
	tbl.SetSelectedRows(0)                   // d
	key(tbl, render.KeyDown, modPrimaryOnly) // cursor on b, not selected
	tbl.SortBy(0, true)                      // a b c d
	if tbl.Selected() < 0 || tbl.Row(tbl.Selected())[0] != "b" {
		t.Errorf("the cursor should stay on b, got row %d", tbl.Selected())
	}
	rows := tbl.SelectedRows()
	if len(rows) != 1 || tbl.Row(rows[0])[0] != "d" {
		t.Errorf("selection should still be just d, got %v", rows)
	}
}
