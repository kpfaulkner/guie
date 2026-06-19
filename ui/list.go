package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const listRowPad = 5

// List is a vertical list of selectable text rows. It scrolls with the wheel,
// selects on click, supports Up/Down/Enter while focused, and highlights the
// hovered and selected rows. It draws only the visible rows.
type List struct {
	BaseWidget
	items    []string
	selected int // -1 = none
	hoverRow int // -1 = none
	offset   float64
	focused  bool
	onSelect func(int)
	font     render.FontFace
}

// ListOption configures a List.
type ListOption func(*List)

// ListSelected sets the initially selected index.
func ListSelected(i int) ListOption { return func(l *List) { l.selected = i } }

// OnSelect registers a handler called with the newly selected index.
func OnSelect(fn func(int)) ListOption { return func(l *List) { l.onSelect = fn } }

// NewList returns a List of items configured by opts.
func NewList(items []string, opts ...ListOption) *List {
	l := &List{BaseWidget: NewBase(), items: items, selected: -1, hoverRow: -1}
	for _, o := range opts {
		o(l)
	}
	return l
}

// Selected returns the selected index, or -1.
func (l *List) Selected() int { return l.selected }

// SetSelected sets the selection and fires OnSelect if it changed.
func (l *List) SetSelected(i int) {
	if i < 0 || i >= len(l.items) || i == l.selected {
		return
	}
	l.selected = i
	if l.onSelect != nil {
		l.onSelect(i)
	}
}

func (l *List) face() render.FontFace {
	if l.font != nil {
		return l.font
	}
	return l.appTheme().Font
}

// RowHeight returns the pixel height of a single row.
func (l *List) RowHeight() float64 {
	f := l.face()
	if f == nil {
		return 0
	}
	return f.Measure("Ag").H + 2*listRowPad
}

// ContentHeight returns the total height of all rows.
func (l *List) ContentHeight() float64 { return l.RowHeight() * float64(len(l.items)) }

// Focusable reports whether the list can take focus (only when enabled).
func (l *List) Focusable() bool { return l.Enabled() }

// MinSize returns the widest row plus padding, and at least one row tall.
func (l *List) MinSize() geom.Size {
	f := l.face()
	if f == nil {
		return geom.Size{}
	}
	var w float64
	for _, it := range l.items {
		w = maxF(w, f.Measure(it).W)
	}
	return geom.Size{W: w + 2*listRowPad + scrollbarWidth, H: l.RowHeight()}
}

func (l *List) maxOffset() float64 {
	return maxF(0, l.ContentHeight()-l.Bounds().H)
}

func (l *List) clamp() {
	if l.offset < 0 {
		l.offset = 0
	}
	if m := l.maxOffset(); l.offset > m {
		l.offset = m
	}
}

// rowAt returns the row index at absolute y, or -1.
func (l *List) rowAt(y float64) int {
	rh := l.RowHeight()
	if rh <= 0 {
		return -1
	}
	idx := int((y - l.Bounds().Y + l.offset) / rh)
	if idx < 0 || idx >= len(l.items) {
		return -1
	}
	return idx
}

// scrollTo adjusts the offset so row i is fully visible.
func (l *List) scrollTo(i int) {
	rh := l.RowHeight()
	top := float64(i) * rh
	bottom := top + rh
	if top < l.offset {
		l.offset = top
	} else if bottom > l.offset+l.Bounds().H {
		l.offset = bottom - l.Bounds().H
	}
	l.clamp()
}

// Draw paints the background, the visible rows (with hover/selection
// highlights), and a scrollbar thumb when the content overflows.
func (l *List) Draw(canvas render.Canvas) {
	pal := l.appTheme().Palette
	f := l.face()
	if f == nil {
		return
	}
	b := l.Bounds()
	canvas.FillRect(b, pal.Surface)

	rh := l.RowHeight()
	overflow := l.ContentHeight() > b.H
	rowW := b.W
	if overflow {
		rowW -= scrollbarWidth
	}

	canvas.PushClip(geom.Rect{X: b.X, Y: b.Y, W: rowW, H: b.H})
	for i, it := range l.items {
		y := b.Y - l.offset + float64(i)*rh
		if y+rh < b.Y || y > b.Y+b.H {
			continue // not visible
		}
		row := geom.Rect{X: b.X, Y: y, W: rowW, H: rh}
		switch {
		case i == l.selected:
			canvas.FillRect(row, pal.Primary)
		case i == l.hoverRow:
			canvas.FillRect(row, lighten(pal.Surface, 1.25))
		}
		textColor := pal.Text
		if i == l.selected {
			textColor = pal.OnPrimary
		}
		canvas.DrawText(it, geom.Point{X: b.X + listRowPad, Y: y + listRowPad}, f, textColor)
	}
	canvas.PopClip()

	canvas.StrokeRect(b, pal.Border, 1)

	if overflow {
		l.drawScrollbar(canvas, b)
	}
}

func (l *List) drawScrollbar(canvas render.Canvas, b geom.Rect) {
	pal := l.appTheme().Palette
	ch := l.ContentHeight()
	thumbH := maxF(minThumb, b.H*b.H/ch)
	var t float64
	if m := l.maxOffset(); m > 0 {
		t = (l.offset / m) * (b.H - thumbH)
	}
	gutter := geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: b.H}
	canvas.FillRect(gutter, pal.Background)
	canvas.FillRect(geom.Rect{X: gutter.X, Y: b.Y + t, W: scrollbarWidth, H: thumbH}, pal.Accent)
}

// HandleEvent handles hover tracking, clicking, wheel scrolling and keyboard
// navigation.
func (l *List) HandleEvent(ev *Event) bool {
	if !l.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerMove:
		l.hoverRow = l.rowAt(ev.Pos.Y)
		return true
	case EventPointerLeave:
		l.hoverRow = -1
		return true
	case EventClick:
		if i := l.rowAt(ev.Pos.Y); i >= 0 {
			l.SetSelected(i)
		}
		return true
	case EventWheel:
		l.offset -= ev.Wheel.Y * wheelStep
		l.clamp()
		return true
	case EventFocusGained:
		l.focused = true
		return true
	case EventFocusLost:
		l.focused = false
		l.hoverRow = -1
		return true
	case EventKeyDown:
		return l.handleKey(ev.Key)
	}
	return false
}

func (l *List) handleKey(k render.Key) bool {
	switch k {
	case render.KeyDown:
		l.move(1)
	case render.KeyUp:
		l.move(-1)
	case render.KeyHome:
		l.selectAndShow(0)
	case render.KeyEnd:
		l.selectAndShow(len(l.items) - 1)
	case render.KeyEnter:
		if l.selected >= 0 && l.onSelect != nil {
			l.onSelect(l.selected)
		}
	default:
		return false
	}
	return true
}

func (l *List) move(delta int) {
	next := l.selected + delta
	if l.selected < 0 {
		next = 0
	}
	l.selectAndShow(next)
}

func (l *List) selectAndShow(i int) {
	if i < 0 || i >= len(l.items) {
		return
	}
	l.SetSelected(i)
	l.scrollTo(i)
}
