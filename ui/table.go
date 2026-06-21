package ui

import (
	"sort"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const tableCellPad = 6

// Column describes a table column.
type Column struct {
	// Title is the header text.
	Title string
	// Weight is the column's share of the available width (0 is treated as 1,
	// giving equal columns by default).
	Weight int
	// Less compares two cell values from this column for sorting (true if a sorts
	// before b). If nil, cells are compared as strings. Set it for numeric or
	// date columns, e.g. by parsing before comparing.
	Less func(a, b string) bool
	// NoSort disables click-to-sort for this column (its header still reports
	// OnHeaderClick).
	NoSort bool
}

// Table is a grid of string cells with a fixed header row and a scrollable,
// selectable body. Rows are selected by click or Up/Down while focused; the
// body scrolls with the wheel and follows the selection. Column widths are
// distributed across the available width by their weights.
type Table struct {
	BaseWidget
	columns  []Column
	rows     [][]string
	selected int // -1 = none
	hoverRow int // -1 = none
	hoverCol int // header column under the cursor, -1 = none
	offset   float64
	focused  bool

	sortable bool // whether header clicks sort (default true)
	sortCol  int  // column currently sorted by, -1 = none
	sortAsc  bool // sort direction

	onSelect      func(int)
	onHeaderClick func(int)
	font          render.FontFace
}

// NewTable returns a Table with the given columns. Click-to-sort is enabled by
// default (see SetSortable).
func NewTable(columns []Column) *Table {
	return &Table{
		BaseWidget: NewBase(),
		columns:    columns,
		selected:   -1,
		hoverRow:   -1,
		hoverCol:   -1,
		sortable:   true,
		sortCol:    -1,
	}
}

// OnHeaderClick registers a handler invoked with the column index when a header
// cell is clicked (fired whether or not the click also triggers a sort).
func (t *Table) OnHeaderClick(fn func(col int)) { t.onHeaderClick = fn }

// SetSortable enables or disables built-in click-to-sort. Disable it to handle
// header clicks entirely via OnHeaderClick.
func (t *Table) SetSortable(v bool) { t.sortable = v }

// SortColumn returns the column the table is sorted by (-1 if unsorted) and the
// direction (true = ascending).
func (t *Table) SortColumn() (col int, ascending bool) { return t.sortCol, t.sortAsc }

// SortBy sorts the rows by column col in the given direction and shows the
// indicator. It is a no-op for an out-of-range column.
func (t *Table) SortBy(col int, ascending bool) {
	if col < 0 || col >= len(t.columns) {
		return
	}
	t.sortCol, t.sortAsc = col, ascending
	t.applySort()
}

// headerColAt returns the header column index containing pos, or -1 if pos is
// not in the header row.
func (t *Table) headerColAt(pos geom.Point) int {
	b := t.Bounds()
	if pos.Y < b.Y || pos.Y >= b.Y+t.rowHeight() {
		return -1
	}
	widths := t.colWidths()
	xs := t.colX(widths)
	for c := range xs {
		if pos.X >= xs[c] && pos.X < xs[c]+widths[c] {
			return c
		}
	}
	return -1
}

// toggleSort sorts by column c, flipping direction if it is already the sort
// column (otherwise sorts ascending).
func (t *Table) toggleSort(c int) {
	if c == t.sortCol {
		t.sortAsc = !t.sortAsc
	} else {
		t.sortCol, t.sortAsc = c, true
	}
	t.applySort()
}

// applySort stably sorts the rows by the active column (using the column's Less,
// or a string compare), preserving the selected row, and requests a redraw.
func (t *Table) applySort() {
	if t.sortCol < 0 || t.sortCol >= len(t.columns) {
		return
	}
	sel := t.selectedRow()
	c := t.sortCol
	less := t.columns[c].Less
	sort.SliceStable(t.rows, func(i, j int) bool {
		a, b := cellOf(t.rows[i], c), cellOf(t.rows[j], c)
		if !t.sortAsc {
			a, b = b, a // reverse the comparison for descending order
		}
		if less != nil {
			return less(a, b)
		}
		return a < b
	})
	t.restoreSelection(sel)
	t.clamp()
	t.Invalidate()
}

// cellOf returns row's cell c, or "" if the row is short.
func cellOf(row []string, c int) string {
	if c < len(row) {
		return row[c]
	}
	return ""
}

// selectedRow returns the currently selected row, or nil.
func (t *Table) selectedRow() []string {
	if t.selected >= 0 && t.selected < len(t.rows) {
		return t.rows[t.selected]
	}
	return nil
}

// restoreSelection relocates sel by row identity after a reorder, updating the
// selection index, or clears it if sel is gone.
func (t *Table) restoreSelection(sel []string) {
	if sel == nil {
		return
	}
	for i, row := range t.rows {
		if sameRow(row, sel) {
			t.selected = i
			return
		}
	}
	t.selected = -1
}

// sameRow reports whether a and b are the same underlying slice.
func sameRow(a, b []string) bool {
	return len(a) > 0 && len(b) > 0 && &a[0] == &b[0]
}

// OnSelect registers the handler invoked with the newly selected row index.
func (t *Table) OnSelect(fn func(int)) { t.onSelect = fn }

// SetRows replaces all rows and clears the selection.
func (t *Table) SetRows(rows [][]string) {
	t.rows = rows
	t.selected = -1
	t.offset = 0
	t.Invalidate()
}

// AddRow appends a row of cells.
func (t *Table) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
	t.Invalidate()
}

// Selected returns the selected row index, or -1.
func (t *Table) Selected() int { return t.selected }

// RowCount returns the number of rows.
func (t *Table) RowCount() int { return len(t.rows) }

// Row returns the cells of row i in the table's current (possibly sorted) order,
// or nil if i is out of range. Use it instead of indexing your own data, since
// sorting reorders the table's rows independently. The returned slice is the
// table's own; do not mutate it.
func (t *Table) Row(i int) []string {
	if i < 0 || i >= len(t.rows) {
		return nil
	}
	return t.rows[i]
}

// SetSelected sets the selection and fires OnRowSelect if it changed.
func (t *Table) SetSelected(i int) {
	if i < 0 || i >= len(t.rows) || i == t.selected {
		return
	}
	t.selected = i
	if t.onSelect != nil {
		t.onSelect(i)
	}
}

func (t *Table) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
}

// SetFont overrides the table's font face (nil falls back to the theme font).
func (t *Table) SetFont(f render.FontFace) {
	t.font = f
	t.Invalidate()
}

// Focusable reports whether the table can take focus (only when enabled).
func (t *Table) Focusable() bool { return t.Enabled() }

func (t *Table) rowHeight() float64 {
	f := t.face()
	if f == nil {
		return 0
	}
	return f.Measure("Ag").H + 2*tableCellPad
}

func (t *Table) contentHeight() float64 { return t.rowHeight() * float64(len(t.rows)) }

// bodyHeight is the height available to rows, below the header.
func (t *Table) bodyHeight() float64 { return t.Bounds().H - t.rowHeight() }

func (t *Table) overflow() bool { return t.contentHeight() > t.bodyHeight() }

func (t *Table) maxOffset() float64 { return maxF(0, t.contentHeight()-t.bodyHeight()) }

func (t *Table) clamp() {
	if t.offset < 0 {
		t.offset = 0
	}
	if m := t.maxOffset(); t.offset > m {
		t.offset = m
	}
}

// colWidths returns the resolved pixel width of each column.
func (t *Table) colWidths() []float64 {
	w := t.Bounds().W
	if t.overflow() {
		w -= scrollbarWidth
	}
	weights := make([]int, len(t.columns))
	for i, c := range t.columns {
		if c.Weight <= 0 {
			weights[i] = 1
		} else {
			weights[i] = c.Weight
		}
	}
	return distributeTracks(len(t.columns), w, 0, weights)
}

// colX returns the left edge x of each column.
func (t *Table) colX(widths []float64) []float64 {
	xs := make([]float64, len(widths))
	x := t.Bounds().X
	for i, cw := range widths {
		xs[i] = x
		x += cw
	}
	return xs
}

// rowAt returns the body row index at absolute y, or -1 (header or out of range).
func (t *Table) rowAt(y float64) int {
	rh := t.rowHeight()
	if rh <= 0 {
		return -1
	}
	bodyTop := t.Bounds().Y + rh
	if y < bodyTop {
		return -1
	}
	idx := int((y - bodyTop + t.offset) / rh)
	if idx < 0 || idx >= len(t.rows) {
		return -1
	}
	return idx
}

func (t *Table) scrollTo(i int) {
	rh := t.rowHeight()
	top := float64(i) * rh
	bottom := top + rh
	if top < t.offset {
		t.offset = top
	} else if bottom > t.offset+t.bodyHeight() {
		t.offset = bottom - t.bodyHeight()
	}
	t.clamp()
}

// MinSize fits the header titles across the columns and leaves room for the
// header plus one body row.
func (t *Table) MinSize() geom.Size {
	f := t.face()
	if f == nil {
		return geom.Size{}
	}
	var w float64
	for _, c := range t.columns {
		w += f.Measure(c.Title).W + 2*tableCellPad
	}
	return geom.Size{W: w, H: t.rowHeight() * 2}
}

// Draw paints the header, the visible body rows (with selection/hover
// highlights), column separators and a scrollbar thumb when needed.
func (t *Table) Draw(canvas render.Canvas) {
	f := t.face()
	if f == nil {
		return
	}
	b := t.Bounds()
	rh := t.rowHeight()
	widths := t.colWidths()
	xs := t.colX(widths)

	canvas.FillRect(b, t.ColourOf(RoleSurface))

	// Header row.
	header := geom.Rect{X: b.X, Y: b.Y, W: b.W, H: rh}
	canvas.FillRect(header, lighten(t.ColourOf(RoleSurface), 1.2))
	for c, col := range t.columns {
		cell := geom.Rect{X: xs[c], Y: b.Y, W: widths[c], H: rh}
		if c == t.hoverCol {
			canvas.FillRect(cell, lighten(t.ColourOf(RoleSurface), 1.3))
		}
		canvas.PushClip(cell)
		canvas.DrawText(col.Title, geom.Point{X: xs[c] + tableCellPad, Y: vCenterY(f, b.Y, rh)}, f, t.ColourOf(RoleText))
		if c == t.sortCol {
			t.drawSortIndicator(canvas, cell, t.sortAsc)
		}
		canvas.PopClip()
	}
	canvas.DrawLine(geom.Point{X: b.X, Y: b.Y + rh}, geom.Point{X: b.X + b.W, Y: b.Y + rh}, t.ColourOf(RoleBorder), 1)

	// Body rows.
	bodyTop := b.Y + rh
	body := geom.Rect{X: b.X, Y: bodyTop, W: b.W, H: b.H - rh}
	canvas.PushClip(body)
	for i, row := range t.rows {
		y := bodyTop - t.offset + float64(i)*rh
		if y+rh < bodyTop || y > b.Y+b.H {
			continue
		}
		rowRect := geom.Rect{X: b.X, Y: y, W: b.W, H: rh}
		textColour := t.ColourOf(RoleText)
		switch {
		case i == t.selected:
			canvas.FillRect(rowRect, t.ColourOf(RolePrimary))
			textColour = t.ColourOf(RoleOnPrimary)
		case i == t.hoverRow:
			canvas.FillRect(rowRect, lighten(t.ColourOf(RoleSurface), 1.25))
		}
		for c := range t.columns {
			if c >= len(row) {
				break
			}
			cell := geom.Rect{X: xs[c], Y: y, W: widths[c], H: rh}
			canvas.PushClip(cell)
			canvas.DrawText(row[c], geom.Point{X: xs[c] + tableCellPad, Y: vCenterY(f, y, rh)}, f, textColour)
			canvas.PopClip()
		}
	}
	canvas.PopClip()

	// Column separators (skip the left edge).
	for c := 1; c < len(xs); c++ {
		canvas.DrawLine(geom.Point{X: xs[c], Y: b.Y}, geom.Point{X: xs[c], Y: b.Y + b.H}, t.ColourOf(RoleBorder), 1)
	}

	canvas.StrokeRect(b, t.ColourOf(RoleBorder), 1)

	if t.overflow() {
		t.drawScrollbar(canvas, b, bodyTop)
	}
}

// drawSortIndicator draws a small up (ascending) or down (descending) chevron at
// the right edge of a header cell.
func (t *Table) drawSortIndicator(canvas render.Canvas, cell geom.Rect, asc bool) {
	cx := cell.X + cell.W - tableCellPad - 4
	cy := cell.Y + cell.H/2
	const s = 3.0
	col := t.ColourOf(RoleAccent)
	if asc {
		canvas.DrawLine(geom.Point{X: cx - s, Y: cy + s/2}, geom.Point{X: cx, Y: cy - s/2}, col, 1.5)
		canvas.DrawLine(geom.Point{X: cx + s, Y: cy + s/2}, geom.Point{X: cx, Y: cy - s/2}, col, 1.5)
	} else {
		canvas.DrawLine(geom.Point{X: cx - s, Y: cy - s/2}, geom.Point{X: cx, Y: cy + s/2}, col, 1.5)
		canvas.DrawLine(geom.Point{X: cx + s, Y: cy - s/2}, geom.Point{X: cx, Y: cy + s/2}, col, 1.5)
	}
}

func (t *Table) drawScrollbar(canvas render.Canvas, b geom.Rect, bodyTop float64) {
	bh := t.bodyHeight()
	ch := t.contentHeight()
	thumbH := maxF(minThumb, bh*bh/ch)
	var off float64
	if m := t.maxOffset(); m > 0 {
		off = (t.offset / m) * (bh - thumbH)
	}
	x := b.X + b.W - scrollbarWidth
	canvas.FillRect(geom.Rect{X: x, Y: bodyTop, W: scrollbarWidth, H: bh}, t.ColourOf(RoleBackground))
	canvas.FillRect(geom.Rect{X: x, Y: bodyTop + off, W: scrollbarWidth, H: thumbH}, t.ColourOf(RoleAccent))
}

// HandleEvent handles hover, row selection, wheel scrolling and keyboard
// navigation.
func (t *Table) HandleEvent(ev *Event) bool {
	if !t.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerMove:
		t.hoverRow = t.rowAt(ev.Pos.Y)
		t.hoverCol = t.headerColAt(ev.Pos)
		return true
	case EventPointerLeave:
		t.hoverRow = -1
		t.hoverCol = -1
		return true
	case EventClick:
		if c := t.headerColAt(ev.Pos); c >= 0 {
			if t.onHeaderClick != nil {
				t.onHeaderClick(c)
			}
			if t.sortable && !t.columns[c].NoSort {
				t.toggleSort(c)
			}
			return true
		}
		if i := t.rowAt(ev.Pos.Y); i >= 0 {
			t.SetSelected(i)
		}
		return true
	case EventWheel:
		t.offset -= ev.Wheel.Y * wheelStep
		t.clamp()
		return true
	case EventFocusGained:
		t.focused = true
		return true
	case EventFocusLost:
		t.focused = false
		t.hoverRow = -1
		return true
	case EventKeyDown:
		return t.handleKey(ev.Key)
	}
	return false
}

func (t *Table) handleKey(k render.Key) bool {
	switch k {
	case render.KeyDown:
		t.moveSelection(1)
	case render.KeyUp:
		t.moveSelection(-1)
	case render.KeyHome:
		t.selectAndShow(0)
	case render.KeyEnd:
		t.selectAndShow(len(t.rows) - 1)
	case render.KeyEnter:
		if t.selected >= 0 && t.onSelect != nil {
			t.onSelect(t.selected)
		}
	default:
		return false
	}
	return true
}

func (t *Table) moveSelection(delta int) {
	next := t.selected + delta
	if t.selected < 0 {
		next = 0
	}
	t.selectAndShow(next)
}

func (t *Table) selectAndShow(i int) {
	if i < 0 || i >= len(t.rows) {
		return
	}
	t.SetSelected(i)
	t.scrollTo(i)
}
