package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const tableCellPad = 6

// Column describes a table column: a header title and a width weight (its share
// of the available width; 0 is treated as 1, giving equal columns by default).
type Column struct {
	Title  string
	Weight int
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
	offset   float64
	focused  bool
	onSelect func(int)
	font     render.FontFace
}

// TableOption configures a Table.
type TableOption func(*Table)

// OnRowSelect registers a handler called with the newly selected row index.
func OnRowSelect(fn func(int)) TableOption { return func(t *Table) { t.onSelect = fn } }

// NewTable returns a Table with the given columns, configured by opts.
func NewTable(columns []Column, opts ...TableOption) *Table {
	t := &Table{BaseWidget: NewBase(), columns: columns, selected: -1, hoverRow: -1}
	for _, o := range opts {
		o(t)
	}
	return t
}

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

	canvas.FillRect(b, t.ColorOf(RoleSurface))

	// Header row.
	header := geom.Rect{X: b.X, Y: b.Y, W: b.W, H: rh}
	canvas.FillRect(header, lighten(t.ColorOf(RoleSurface), 1.2))
	for c, col := range t.columns {
		cell := geom.Rect{X: xs[c], Y: b.Y, W: widths[c], H: rh}
		canvas.PushClip(cell)
		canvas.DrawText(col.Title, geom.Point{X: xs[c] + tableCellPad, Y: b.Y + tableCellPad}, f, t.ColorOf(RoleText))
		canvas.PopClip()
	}
	canvas.DrawLine(geom.Point{X: b.X, Y: b.Y + rh}, geom.Point{X: b.X + b.W, Y: b.Y + rh}, t.ColorOf(RoleBorder), 1)

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
		textColor := t.ColorOf(RoleText)
		switch {
		case i == t.selected:
			canvas.FillRect(rowRect, t.ColorOf(RolePrimary))
			textColor = t.ColorOf(RoleOnPrimary)
		case i == t.hoverRow:
			canvas.FillRect(rowRect, lighten(t.ColorOf(RoleSurface), 1.25))
		}
		for c := range t.columns {
			if c >= len(row) {
				break
			}
			cell := geom.Rect{X: xs[c], Y: y, W: widths[c], H: rh}
			canvas.PushClip(cell)
			canvas.DrawText(row[c], geom.Point{X: xs[c] + tableCellPad, Y: y + tableCellPad}, f, textColor)
			canvas.PopClip()
		}
	}
	canvas.PopClip()

	// Column separators (skip the left edge).
	for c := 1; c < len(xs); c++ {
		canvas.DrawLine(geom.Point{X: xs[c], Y: b.Y}, geom.Point{X: xs[c], Y: b.Y + b.H}, t.ColorOf(RoleBorder), 1)
	}

	canvas.StrokeRect(b, t.ColorOf(RoleBorder), 1)

	if t.overflow() {
		t.drawScrollbar(canvas, b, bodyTop)
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
	canvas.FillRect(geom.Rect{X: x, Y: bodyTop, W: scrollbarWidth, H: bh}, t.ColorOf(RoleBackground))
	canvas.FillRect(geom.Rect{X: x, Y: bodyTop + off, W: scrollbarWidth, H: thumbH}, t.ColorOf(RoleAccent))
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
		return true
	case EventPointerLeave:
		t.hoverRow = -1
		return true
	case EventClick:
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
