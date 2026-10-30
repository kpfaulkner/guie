package ui

import "github.com/kpfaulkner/guie/render"

// Multiple selection for Table. It is off by default, and a Table that never
// calls SetMultiSelect behaves exactly as before.
//
// The model is a set of selected rows plus two single rows: the active row
// (Selected), which is where the keyboard cursor is and what OnSelect and
// OnActivate report, and the anchor, the fixed end of a Shift range. Plain
// clicks and arrow keys collapse the set to one row; Ctrl/Cmd+click toggles a
// row; Shift+click and Shift+arrows select the range from the anchor to the
// active row (with Ctrl/Cmd held too the range is added to what is selected);
// Ctrl/Cmd+A selects every row. For the keyboard alone: Ctrl/Cmd+Up, Down, Home
// and End move the active row without touching the selection, Space toggles the
// active row, and Ctrl/Cmd+Shift+arrows add the range from the anchor.

// SetMultiSelect turns multiple selection on or off (default off). Call it
// before the table is in use: turning it on keeps the active row selected, and
// turning it off collapses the selection to the active row. It fires no
// callbacks.
func (t *Table) SetMultiSelect(on bool) {
	if t.multi == on {
		return
	}
	t.multi = on
	if on {
		t.sel, t.anchor = nil, t.selected
		if t.selected >= 0 && t.selected < len(t.rows) {
			t.sel = rowSet(len(t.rows), t.selected)
		}
	} else {
		t.sel, t.anchor = nil, -1
	}
	t.Invalidate()
}

// MultiSelect reports whether multiple selection is on.
func (t *Table) MultiSelect() bool { return t.multi }

// OnSelectionChange registers a handler called with the selected rows, in
// ascending order, whenever the set of selected rows changes by user action or
// by SetSelected and SetSelectedRows. Like OnSelect it is not called when
// SetRows clears the selection.
func (t *Table) OnSelectionChange(fn func(rows []int)) { t.onSelectionChange = fn }

// SelectedRows returns the selected row indexes in ascending order, or nil if
// none. Without multiple selection that is just the Selected row. The indexes
// are displayed rows, as for Row.
func (t *Table) SelectedRows() []int {
	if !t.multi {
		if t.selected >= 0 && t.selected < len(t.rows) {
			return []int{t.selected}
		}
		return nil
	}
	var out []int
	for i, on := range t.sel {
		if on && i < len(t.rows) {
			out = append(out, i)
		}
	}
	return out
}

// SetSelectedRows selects exactly these rows, ignoring any out of range. The
// last valid one becomes the active row and the anchor. Without multiple
// selection only that last row is selected. Passing no valid row clears the
// selection.
func (t *Table) SetSelectedRows(rows ...int) {
	sel := make([]bool, len(t.rows))
	active := -1
	for _, r := range rows {
		if r >= 0 && r < len(t.rows) {
			sel[r] = true
			active = r
		}
	}
	if !t.multi {
		if active < 0 {
			t.clearSelection()
		} else {
			t.SetSelected(active)
		}
		return
	}
	t.commit(sel, active, active)
}

// clearSelection drops the selection without firing OnSelect, which has no way
// to say "nothing"; OnSelectionChange is told.
func (t *Table) clearSelection() {
	had := len(t.SelectedRows()) > 0
	t.selected, t.anchor, t.sel = -1, -1, nil
	t.Invalidate()
	if had && t.onSelectionChange != nil {
		t.onSelectionChange(nil)
	}
}

// isSelected reports whether row i is highlighted as selected.
func (t *Table) isSelected(i int) bool {
	if t.multi {
		return i >= 0 && i < len(t.sel) && t.sel[i]
	}
	return i == t.selected
}

// rowSet returns a set of n rows with only row i selected.
func rowSet(n, i int) []bool {
	s := make([]bool, n)
	s[i] = true
	return s
}

// commit replaces the multi-select state and tells the callbacks what changed:
// OnSelect if the active row moved, OnSelectionChange if the set changed.
func (t *Table) commit(sel []bool, active, anchor int) {
	before := t.SelectedRows()
	prevActive := t.selected
	wasSelected := t.isSelected(active) // false for no row
	t.sel, t.selected, t.anchor = sel, active, anchor
	t.Invalidate()
	// OnSelect is for the active row becoming selected, by moving onto it or by
	// toggling it on. A toggle that deselects it moves the cursor but selects
	// nothing, and select-all leaves a row that was already selected alone.
	if active >= 0 && t.isSelected(active) && (active != prevActive || !wasSelected) && t.onSelect != nil {
		t.onSelect(active)
	}
	if after := t.SelectedRows(); !sameInts(before, after) && t.onSelectionChange != nil {
		t.onSelectionChange(after)
	}
}

func sameInts(a, b []int) bool {
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

// clickRow applies a click on row i, honouring Shift and Ctrl/Cmd in multiple
// selection.
func (t *Table) clickRow(i int, mods render.ModifierSet) {
	switch {
	case !t.multi:
		t.SetSelected(i)
	case mods.Has(render.ModShift):
		t.extendTo(i, mods.Has(render.ModPrimary))
	case mods.Has(render.ModPrimary):
		sel := t.copySel()
		sel[i] = !sel[i]
		t.commit(sel, i, i)
	default:
		t.SetSelected(i)
	}
}

// copySel returns a copy of the selection sized to the rows.
func (t *Table) copySel() []bool {
	sel := make([]bool, len(t.rows))
	copy(sel, t.sel)
	return sel
}

// extendTo selects the rows from the anchor to row i and makes i the active
// row. With add the range joins the current selection; without, it replaces it.
// With no anchor yet, i is the anchor.
func (t *Table) extendTo(i int, add bool) {
	if i < 0 || i >= len(t.rows) {
		return
	}
	anchor := t.anchor
	if anchor < 0 || anchor >= len(t.rows) {
		anchor = i
	}
	sel := make([]bool, len(t.rows))
	if add {
		copy(sel, t.sel)
	}
	lo, hi := anchor, i
	if lo > hi {
		lo, hi = hi, lo
	}
	for r := lo; r <= hi; r++ {
		sel[r] = true
	}
	t.commit(sel, i, anchor)
}

// navMode says what an arrow, Home or End key does to the selection.
type navMode int

const (
	navPlain  navMode = iota // move and select just that row
	navExtend                // Shift: select the range from the anchor
	navAdd                   // Ctrl/Cmd+Shift: add the range from the anchor
	navCursor                // Ctrl/Cmd: move the cursor, leave the selection
)

// navModeFor maps the held modifiers to a navMode. Without multiple selection
// every key is navPlain, so a single-select table never sees the others.
func (t *Table) navModeFor(mods render.ModifierSet) navMode {
	if !t.multi {
		return navPlain
	}
	shift, primary := mods.Has(render.ModShift), mods.Has(render.ModPrimary)
	switch {
	case shift && primary:
		return navAdd
	case primary:
		return navCursor
	case shift:
		return navExtend
	}
	return navPlain
}

// navigate handles Up, Down, Home and End. Stepping off either end does nothing.
// From no active row, Up and Down both start at the first row.
func (t *Table) navigate(k render.Key, mode navMode) {
	n := len(t.rows)
	if n == 0 {
		return
	}
	target := 0
	switch k {
	case render.KeyDown:
		if t.selected >= 0 {
			target = t.selected + 1
		}
	case render.KeyUp:
		if t.selected >= 0 {
			target = t.selected - 1
		}
	case render.KeyEnd:
		target = n - 1
	}
	if target < 0 || target >= n {
		return
	}
	switch mode {
	case navExtend:
		t.extendTo(target, false)
		t.scrollTo(target)
	case navAdd:
		t.extendTo(target, true)
		t.scrollTo(target)
	case navCursor:
		t.moveCursor(target)
	default:
		t.selectAndShow(target)
	}
}

// moveCursor makes row i the active row and scrolls to it, leaving the selection
// as it is. Nothing is reported: no row was selected or deselected.
func (t *Table) moveCursor(i int) {
	t.selected = i
	t.scrollTo(i)
	t.Invalidate()
}

// toggleActive flips the selection of the active row, which is how Space works.
// With no active row it selects the first one.
func (t *Table) toggleActive() {
	if len(t.rows) == 0 {
		return
	}
	if t.selected < 0 || t.selected >= len(t.rows) {
		t.commit(rowSet(len(t.rows), 0), 0, 0)
		t.scrollTo(0)
		return
	}
	sel := t.copySel()
	sel[t.selected] = !sel[t.selected]
	t.commit(sel, t.selected, t.selected)
	t.scrollTo(t.selected)
}

// selectAll selects every row, leaving the active row and anchor where they are.
func (t *Table) selectAll() {
	sel := make([]bool, len(t.rows))
	for i := range sel {
		sel[i] = true
	}
	t.commit(sel, t.selected, t.anchor)
}

// pickedRows returns the selected rows by identity, so the selection can follow
// them through a re-sort. Rows with no cells have no identity and are dropped.
func (t *Table) pickedRows() map[*string]bool {
	picked := map[*string]bool{}
	for i, on := range t.sel {
		if on && i < len(t.rows) && len(t.rows[i]) > 0 {
			picked[&t.rows[i][0]] = true
		}
	}
	return picked
}

// restoreMulti rebuilds the selection and anchor after the rows were reordered,
// from the identities pickedRows and the anchor row recorded beforehand.
func (t *Table) restoreMulti(picked map[*string]bool, anchorRow []string) {
	sel := make([]bool, len(t.rows))
	anchor := -1
	for i, row := range t.rows {
		if len(row) > 0 && picked[&row[0]] {
			sel[i] = true
		}
		if sameRow(row, anchorRow) {
			anchor = i
		}
	}
	t.sel, t.anchor = sel, anchor
}
