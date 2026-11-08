package ui

import "slices"

// This file holds Table methods for tables whose rows change while they are on
// screen, such as a live log: removing rows without disturbing the user, and
// following the end of the list.

// RemoveRow removes row i, in the table's current (possibly sorted) order. An
// out-of-range i is ignored.
//
// The selection stays on the same rows: indexes after i shift down by one. If
// row i itself was selected it is simply no longer selected; OnSelectionChange
// is told, OnSelect is not (it has no way to say "nothing"). The rows on
// screen stay where they are: removing a row above the view scrolls up by one
// row to compensate, so the content does not jump.
func (t *Table) RemoveRow(i int) {
	if i < 0 || i >= len(t.rows) {
		return
	}
	before := t.SelectedRows()

	t.rows = slices.Delete(t.rows, i, i+1)
	if t.sel != nil && i < len(t.sel) {
		t.sel = slices.Delete(t.sel, i, i+1)
	}
	t.selected = shiftAfterRemove(t.selected, i)
	t.anchor = shiftAfterRemove(t.anchor, i)
	t.hoverRow = -1 // recomputed on the next pointer move

	if rh := t.rowHeight(); rh > 0 && float64(i)*rh < t.offset {
		t.offset -= rh
	}
	t.clamp()
	t.Invalidate()

	if after := t.SelectedRows(); !sameInts(before, after) && t.onSelectionChange != nil {
		t.onSelectionChange(after)
	}
}

// shiftAfterRemove maps a row index to its new value once row removed is gone:
// -1 if it was that row, one less if it was after it.
func shiftAfterRemove(idx, removed int) int {
	switch {
	case idx == removed:
		return -1
	case idx > removed:
		return idx - 1
	}
	return idx
}

// ClearSelection deselects every row. OnSelectionChange is told if anything was
// selected; OnSelect is not.
func (t *Table) ClearSelection() { t.clearSelection() }

// ScrollToRow scrolls the least distance needed to bring row i fully into view.
// ScrollToRow(RowCount()-1) scrolls to the bottom. An out-of-range i is ignored.
// Before the table has been laid out it has no size, so there is nothing to
// scroll yet.
func (t *Table) ScrollToRow(i int) {
	if i < 0 || i >= len(t.rows) || t.Bounds().H <= 0 {
		return
	}
	t.scrollTo(i)
	t.Invalidate()
}

// ScrolledToBottom reports whether the view shows the last row, including when
// every row fits. A table that has not been laid out yet counts as at the
// bottom. Use it to follow new rows only while the user has not scrolled up:
//
//	follow := tbl.ScrolledToBottom()
//	tbl.AddRow(cells...)
//	if follow {
//		tbl.ScrollToRow(tbl.RowCount() - 1)
//	}
func (t *Table) ScrolledToBottom() bool {
	if t.Bounds().H <= 0 {
		return true
	}
	return t.offset >= t.maxOffset()-0.5
}
