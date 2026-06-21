package ui

import (
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	dateFieldIcon     = 13 // size of the little calendar glyph
	dateFieldVPadding = 5
	dateFieldHPadding = 8
)

// dateFieldSample is a wide date used only to size the field so any formatted
// date fits.
var dateFieldSample = time.Date(2006, time.September, 30, 0, 0, 0, 0, time.UTC)

// DateField is a collapsed date control: it shows the selected date (or a
// placeholder) and opens a DatePicker calendar in a popup below it when clicked.
// Choosing a day updates the value, fires OnChange and closes the popup; an
// outside click or Escape also closes it. It is the popup counterpart to the
// inline DatePicker.
type DateField struct {
	BaseWidget
	value        time.Time
	hasValue     bool
	placeholder  string
	format       string
	firstWeekday time.Weekday

	hover   bool
	focused bool
	open    bool
	popup   *Popup

	onChange func(time.Time)
	font     render.FontFace
}

// DateFieldOption configures a DateField.
type DateFieldOption func(*DateField)

// DateFieldValue sets the initial date (and marks the field as having a value).
func DateFieldValue(t time.Time) DateFieldOption {
	return func(f *DateField) { f.value, f.hasValue = dayOf(t), true }
}

// DateFieldFormat sets the display layout (a Go reference-time layout, e.g.
// "2006-01-02"). The default is "2 Jan 2006".
func DateFieldFormat(layout string) DateFieldOption {
	return func(f *DateField) {
		if layout != "" {
			f.format = layout
		}
	}
}

// DateFieldPlaceholder sets the text shown when no date is selected.
func DateFieldPlaceholder(s string) DateFieldOption {
	return func(f *DateField) { f.placeholder = s }
}

// DateFieldFirstWeekday sets the calendar's leftmost weekday (default Sunday).
func DateFieldFirstWeekday(w time.Weekday) DateFieldOption {
	return func(f *DateField) { f.firstWeekday = w }
}

// NewDateField returns a DateField with no date selected (showing its
// placeholder) unless DateFieldValue is given.
func NewDateField(opts ...DateFieldOption) *DateField {
	f := &DateField{
		BaseWidget:   NewBase(),
		placeholder:  "Select date...",
		format:       "2 Jan 2006",
		firstWeekday: time.Sunday,
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

// OnChange registers the handler invoked when the selected date changes.
func (f *DateField) OnChange(fn func(time.Time)) { f.onChange = fn }

// Value returns the selected date and whether a date has been set.
func (f *DateField) Value() (time.Time, bool) { return f.value, f.hasValue }

// SetValue selects t (date-only), marking the field set, and fires OnChange if
// the day changed.
func (f *DateField) SetValue(t time.Time) {
	nt := dayOf(t)
	if f.hasValue && sameDay(nt, f.value) {
		return
	}
	f.value = nt
	f.hasValue = true
	if f.onChange != nil {
		f.onChange(nt)
	}
	f.Invalidate()
}

func (f *DateField) face() render.FontFace {
	if f.font != nil {
		return f.font
	}
	return f.appTheme().Font
}

// SetFont overrides the field's font face (nil falls back to the theme font).
func (f *DateField) SetFont(face render.FontFace) {
	f.font = face
	f.Invalidate()
}

// Focusable reports whether the field can take focus (only when enabled).
func (f *DateField) Focusable() bool { return f.Enabled() }

func (f *DateField) label() string {
	if f.hasValue {
		return f.value.Format(f.format)
	}
	return f.placeholder
}

// MinSize fits the placeholder or a wide sample date, plus the calendar glyph.
func (f *DateField) MinSize() geom.Size {
	face := f.face()
	if face == nil {
		return geom.Size{}
	}
	w := maxF(face.Measure(f.placeholder).W, face.Measure(dateFieldSample.Format(f.format)).W)
	h := face.Measure("Ag").H
	return geom.Size{
		W: w + dateFieldIcon + 2*dateFieldHPadding,
		H: h + 2*dateFieldVPadding,
	}
}

// Draw renders the box, the date (or placeholder) and a small calendar glyph,
// accented when focused or open.
func (f *DateField) Draw(canvas render.Canvas) {
	face := f.face()
	if face == nil {
		return
	}
	b := f.Bounds()
	rad := f.cornerRadius()
	canvas.FillRoundRect(b, rad, f.ColourOf(RoleSurface))
	border := f.ColourOf(RoleBorder)
	if f.focused || f.open {
		border = f.ColourOf(RoleAccent)
	}
	canvas.StrokeRoundRect(b, rad, border, 1)

	textColour := f.ColourOf(RoleText)
	if !f.hasValue {
		textColour = f.ColourOf(RoleTextMuted)
	}
	canvas.DrawText(f.label(), geom.Point{X: b.X + dateFieldHPadding, Y: vCenterY(face, b.Y, b.H)}, face, textColour)

	f.drawCalendarGlyph(canvas, b)
}

// drawCalendarGlyph draws a small calendar icon in the right padding.
func (f *DateField) drawCalendarGlyph(canvas render.Canvas, b geom.Rect) {
	col := f.ColourOf(RoleText)
	s := float64(dateFieldIcon)
	x := b.X + b.W - dateFieldHPadding - s
	y := b.Y + (b.H-s)/2
	canvas.StrokeRoundRect(geom.Rect{X: x, Y: y, W: s, H: s}, 2, col, 1)
	// Header bar across the top of the calendar.
	canvas.DrawLine(geom.Point{X: x, Y: y + s/3}, geom.Point{X: x + s, Y: y + s/3}, col, 1)
}

// HandleEvent toggles the popup on click/Enter/Space and tracks hover/focus.
func (f *DateField) HandleEvent(ev *Event) bool {
	if !f.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		f.hover = true
		return true
	case EventPointerLeave:
		f.hover = false
		return true
	case EventClick:
		f.toggle()
		return true
	case EventFocusGained:
		f.focused = true
		return true
	case EventFocusLost:
		f.focused = false
		return true
	case EventKeyDown:
		if ev.Key == render.KeySpace || ev.Key == render.KeyEnter {
			f.toggle()
			return true
		}
	}
	return false
}

func (f *DateField) toggle() {
	if f.open {
		f.ctx.close(f.popup)
		return
	}
	f.openCalendar()
}

// openCalendar builds a DatePicker popup positioned directly below the field.
func (f *DateField) openCalendar() {
	if f.ctx == nil {
		return
	}
	opts := []DatePickerOption{DatePickerFirstWeekday(f.firstWeekday)}
	if f.hasValue {
		opts = append(opts, DatePickerValue(f.value))
	}
	cal := NewDatePicker(opts...)
	cal.font = f.face() // resolved font so MinSize works before mount
	cal.OnChange(func(t time.Time) {
		f.SetValue(t)
		f.ctx.close(f.popup)
	})

	b := f.Bounds()
	size := cal.MinSize()
	w := maxF(size.W, b.W)
	bounds := geom.Rect{X: b.X, Y: b.Y + b.H + 2, W: w, H: size.H}

	f.popup = NewPopup(cal, bounds, func() { f.open = false })
	f.ctx.open(f.popup)
	f.open = true
}
