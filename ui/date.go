package ui

import (
	"image/color"
	"strconv"
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// datePadding is the inner padding around the calendar grid.
var datePadding = geom.UniformInsets(6)

// Calendar layout: 8 rows (header, weekday labels, then 6 week rows) by 7 cols.
const (
	dateRows = 8
	dateCols = 7
	dateWeek = 6 // week rows shown
)

// DatePicker is an inline month calendar. It shows one month at a time with a
// header to step months, a weekday row and a 6-week grid of day cells. Click a
// day (including a dimmed adjacent-month day) to select it; the arrow keys move
// the selection by day/week and PageUp/PageDown by month. OnChange fires when
// the selected date changes. It is self-drawn and exposes no child widgets.
type DatePicker struct {
	BaseWidget
	selected     time.Time // selected date (date-only)
	visible      time.Time // first day of the displayed month
	today        time.Time // date highlighted as "today"
	firstWeekday time.Weekday
	focused      bool
	hover        int // hovered week-grid cell index (0..41), -1 for none

	font     render.FontFace
	onChange func(time.Time)
}

// DatePickerOption configures a DatePicker.
type DatePickerOption func(*DatePicker)

// DatePickerValue sets the initially selected date.
func DatePickerValue(t time.Time) DatePickerOption {
	return func(d *DatePicker) { d.selected = dayOf(t) }
}

// DatePickerToday sets the date marked as "today" (defaults to the current
// date). Useful for deterministic tests.
func DatePickerToday(t time.Time) DatePickerOption {
	return func(d *DatePicker) { d.today = dayOf(t) }
}

// DatePickerFirstWeekday sets the leftmost column's weekday (default Sunday).
func DatePickerFirstWeekday(w time.Weekday) DatePickerOption {
	return func(d *DatePicker) { d.firstWeekday = w }
}

// NewDatePicker returns a DatePicker showing the month of the selected date
// (today by default).
func NewDatePicker(opts ...DatePickerOption) *DatePicker {
	now := dayOf(time.Now())
	d := &DatePicker{
		BaseWidget:   NewBase(),
		selected:     now,
		today:        now,
		firstWeekday: time.Sunday,
		hover:        -1,
	}
	for _, o := range opts {
		o(d)
	}
	d.visible = firstOfMonth(d.selected)
	return d
}

// OnChange registers the handler invoked when the selected date changes.
func (d *DatePicker) OnChange(fn func(time.Time)) { d.onChange = fn }

// Value returns the selected date (date-only, midnight).
func (d *DatePicker) Value() time.Time { return d.selected }

// SetValue selects t (date-only), scrolls the view to its month, and fires
// OnChange if the day changed.
func (d *DatePicker) SetValue(t time.Time) {
	nd := dayOf(t)
	d.visible = firstOfMonth(nd)
	if sameDay(nd, d.selected) {
		return
	}
	d.selected = nd
	if d.onChange != nil {
		d.onChange(nd)
	}
	d.Invalidate()
}

// ShowMonth navigates the view to the given month without changing the
// selection.
func (d *DatePicker) ShowMonth(year int, m time.Month) {
	d.visible = time.Date(year, m, 1, 0, 0, 0, 0, time.Local)
	d.Invalidate()
}

func (d *DatePicker) face() render.FontFace {
	if d.font != nil {
		return d.font
	}
	return d.appTheme().Font
}

// SetFont overrides the calendar's font face (nil falls back to the theme font).
func (d *DatePicker) SetFont(f render.FontFace) {
	d.font = f
	d.Invalidate()
}

// Focusable reports whether the calendar can take focus (only when enabled).
func (d *DatePicker) Focusable() bool { return d.Enabled() }

// MinSize returns a calendar sized to fit two-digit days and weekday labels.
func (d *DatePicker) MinSize() geom.Size {
	f := d.face()
	var cell float64 = 22
	if f != nil {
		cell = maxF(f.Measure("30").W, f.Measure("Wd").W) + 10
		lh := f.Measure("Ag").H + 6
		return geom.Size{
			W: cell*dateCols + datePadding.Left + datePadding.Right,
			H: lh*dateRows + datePadding.Top + datePadding.Bottom,
		}
	}
	return geom.Size{W: cell * dateCols, H: cell * dateRows}
}

// grid returns the content origin and per-cell dimensions.
func (d *DatePicker) grid() (origin geom.Point, cellW, rowH float64) {
	inner := d.Bounds().Inset(datePadding)
	return inner.Min(), inner.W / dateCols, inner.H / dateRows
}

// offset is the number of leading cells before day 1 of the visible month.
func (d *DatePicker) offset() int {
	return (int(d.visible.Weekday()) - int(d.firstWeekday) + 7) % 7
}

// dateForCell returns the date shown in week-grid cell i (0..41); it may fall in
// the previous or next month.
func (d *DatePicker) dateForCell(i int) time.Time {
	return d.visible.AddDate(0, 0, i-d.offset())
}

// cellForDate returns the week-grid cell index for date t, or -1 if t is not on
// the current grid.
func (d *DatePicker) cellForDate(t time.Time) int {
	days := int(dayOf(t).Sub(d.visible).Hours() / 24)
	i := days + d.offset()
	if i < 0 || i >= dateWeek*dateCols {
		return -1
	}
	return i
}

// Draw paints the header, weekday labels and the day grid.
func (d *DatePicker) Draw(canvas render.Canvas) {
	f := d.face()
	if f == nil {
		return
	}
	b := d.Bounds()
	canvas.FillRoundRect(b, d.cornerRadius(), d.ColourOf(RoleSurface))
	border := d.ColourOf(RoleBorder)
	if d.focused {
		border = d.ColourOf(RoleAccent)
	}
	canvas.StrokeRoundRect(b, d.cornerRadius(), border, 1)

	origin, cellW, rowH := d.grid()
	textCol := d.ColourOf(RoleText)
	muted := d.ColourOf(RoleTextMuted)

	// Header: ‹ Month Year ›
	d.drawCenteredText(canvas, "‹", origin.X, origin.Y, cellW, rowH, f, textCol)
	d.drawCenteredText(canvas, "›", origin.X+cellW*6, origin.Y, cellW, rowH, f, textCol)
	title := d.visible.Format("January 2006")
	d.drawCenteredText(canvas, title, origin.X+cellW, origin.Y, cellW*5, rowH, f, textCol)

	// Weekday labels.
	for c := 0; c < dateCols; c++ {
		wd := time.Weekday((int(d.firstWeekday) + c) % 7)
		label := wd.String()[:2]
		d.drawCenteredText(canvas, label, origin.X+float64(c)*cellW, origin.Y+rowH, cellW, rowH, f, muted)
	}

	// Day cells.
	for i := 0; i < dateWeek*dateCols; i++ {
		date := d.dateForCell(i)
		row := 2 + i/dateCols
		col := i % dateCols
		x := origin.X + float64(col)*cellW
		y := origin.Y + float64(row)*rowH
		cell := geom.Rect{X: x, Y: y, W: cellW, H: rowH}

		inMonth := date.Month() == d.visible.Month()
		switch {
		case sameDay(date, d.selected):
			canvas.FillRoundRect(cell.Inset(geom.UniformInsets(2)), 4, d.ColourOf(RolePrimary))
		case i == d.hover:
			canvas.FillRoundRect(cell.Inset(geom.UniformInsets(2)), 4, lighten(d.ColourOf(RoleSurface), 1.25))
		}
		if sameDay(date, d.today) && !sameDay(date, d.selected) {
			canvas.StrokeRoundRect(cell.Inset(geom.UniformInsets(2)), 4, d.ColourOf(RoleAccent), 1)
		}

		col2 := textCol
		switch {
		case sameDay(date, d.selected):
			col2 = d.ColourOf(RoleOnPrimary)
		case !inMonth:
			col2 = muted
		}
		d.drawCenteredText(canvas, strconv.Itoa(date.Day()), x, y, cellW, rowH, f, col2)
	}
}

// drawCenteredText draws s horizontally centered within the [x, x+w] span and
// vertically centered in [y, y+h].
func (d *DatePicker) drawCenteredText(canvas render.Canvas, s string, x, y, w, h float64, f render.FontFace, col color.Color) {
	tw := f.Measure(s).W
	canvas.DrawText(s, geom.Point{X: x + (w-tw)/2, Y: vCenterY(f, y, h)}, f, col)
}

// HandleEvent handles month navigation, day selection, hover and keyboard nav.
func (d *DatePicker) HandleEvent(ev *Event) bool {
	if !d.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerLeave:
		d.hover = -1
		return true
	case EventPointerMove:
		d.hover = d.cellAt(ev.Pos)
		return true
	case EventClick:
		d.handleClick(ev.Pos)
		return true
	case EventWheel:
		if ev.Wheel.Y > 0 {
			d.stepMonth(-1)
		} else if ev.Wheel.Y < 0 {
			d.stepMonth(1)
		}
		return true
	case EventFocusGained:
		d.focused = true
		return true
	case EventFocusLost:
		d.focused = false
		d.hover = -1
		return true
	case EventKeyDown:
		return d.handleKey(ev.Key)
	}
	return false
}

// cellAt maps an absolute point to a week-grid cell index (0..41), or -1 if the
// point is in the header/weekday rows or outside the grid.
func (d *DatePicker) cellAt(p geom.Point) int {
	origin, cellW, rowH := d.grid()
	if cellW <= 0 || rowH <= 0 {
		return -1
	}
	col := int((p.X - origin.X) / cellW)
	row := int((p.Y - origin.Y) / rowH)
	if col < 0 || col >= dateCols || row < 2 || row >= dateRows {
		return -1
	}
	return (row-2)*dateCols + col
}

func (d *DatePicker) handleClick(p geom.Point) {
	origin, cellW, rowH := d.grid()
	if rowH <= 0 {
		return
	}
	// Header row: prev/next month on the corner cells.
	if p.Y < origin.Y+rowH {
		switch {
		case p.X < origin.X+cellW:
			d.stepMonth(-1)
		case p.X >= origin.X+cellW*6:
			d.stepMonth(1)
		}
		return
	}
	if i := d.cellAt(p); i >= 0 {
		d.SetValue(d.dateForCell(i))
	}
}

func (d *DatePicker) handleKey(k render.Key) bool {
	switch k {
	case render.KeyLeft:
		d.SetValue(d.selected.AddDate(0, 0, -1))
	case render.KeyRight:
		d.SetValue(d.selected.AddDate(0, 0, 1))
	case render.KeyUp:
		d.SetValue(d.selected.AddDate(0, 0, -7))
	case render.KeyDown:
		d.SetValue(d.selected.AddDate(0, 0, 7))
	case render.KeyPageUp:
		d.SetValue(d.selected.AddDate(0, -1, 0))
	case render.KeyPageDown:
		d.SetValue(d.selected.AddDate(0, 1, 0))
	default:
		return false
	}
	return true
}

// stepMonth moves the view by n months without changing the selection.
func (d *DatePicker) stepMonth(n int) {
	d.visible = d.visible.AddDate(0, n, 0)
	d.hover = -1
	d.Invalidate()
}

// --- date helpers ---

// dayOf truncates t to its calendar date (midnight, same location).
func dayOf(t time.Time) time.Time {
	y, m, dd := t.Date()
	return time.Date(y, m, dd, 0, 0, 0, 0, t.Location())
}

func firstOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
