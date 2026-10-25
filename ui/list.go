package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const listRowPad = 5

// List is a vertical list of selectable text rows. It scrolls with the wheel,
// selects on click, supports Up/Down/Enter while focused, and highlights the
// hovered and selected rows. It draws only the visible rows.
type List struct {
	BaseWidget
	items      []string
	selected   int // -1 = none
	hoverRow   int // -1 = none
	offset     float64
	bar        barDrag // scrollbar thumb-drag state
	offsetX    float64
	barH       barDrag  // horizontal scrollbar thumb-drag state
	hscroll    bool     // rows may be wider than the list, scrolling across
	wStr       []string // items whose widths are cached in wW
	wW         []float64
	wFP        float64 // font fingerprint the cached widths were measured with
	focused    bool
	onSelect   func(int)
	onActivate func(int)
	font       render.FontFace
}

// ListOption configures a List.
type ListOption func(*List)

// ListSelected sets the initially selected index.
func ListSelected(i int) ListOption { return func(l *List) { l.selected = i } }

// NewList returns a List of items configured by opts.
func NewList(items []string, opts ...ListOption) *List {
	l := &List{BaseWidget: NewBase(), items: items, selected: -1, hoverRow: -1}
	for _, o := range opts {
		o(l)
	}
	return l
}

// OnSelect registers the handler invoked with the newly selected index.
func (l *List) OnSelect(fn func(int)) { l.onSelect = fn }

// OnActivate registers the handler invoked with a row index when that row is
// activated: double-clicked, or Enter pressed while it is selected. A double-click
// also selects the row (via its first click), so OnSelect has already fired.
func (l *List) OnActivate(fn func(int)) { l.onActivate = fn }

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

// SetFont overrides the list's font face (nil falls back to the theme font).
func (l *List) SetFont(f render.FontFace) {
	l.font = f
	l.Invalidate()
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
	if l.hscroll {
		// Like its height, the width is the parent's to give: wide rows scroll.
		return geom.Size{W: minViewport + scrollbarWidth, H: l.RowHeight()}
	}
	var w float64
	for _, it := range l.items {
		w = maxF(w, f.Measure(it).W)
	}
	return geom.Size{W: w + 2*listRowPad + scrollbarWidth, H: l.RowHeight()}
}

func (l *List) maxOffset() float64 {
	return maxF(0, l.ContentHeight()-l.geomInfo().viewH)
}

func (l *List) clamp() {
	if l.offset < 0 {
		l.offset = 0
	}
	if m := l.maxOffset(); l.offset > m {
		l.offset = m
	}
	l.offsetX = minF(maxF(l.offsetX, 0), l.maxOffsetX())
}

// rowAt returns the row index at absolute y, or -1.
func (l *List) rowAt(y float64) int {
	rh := l.RowHeight()
	if rh <= 0 {
		return -1
	}
	if y >= l.Bounds().Y+l.geomInfo().viewH {
		return -1 // the horizontal bar, if any
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
	} else if viewH := l.geomInfo().viewH; bottom > l.offset+viewH {
		l.offset = bottom - viewH
	}
	l.clamp()
}

// Draw paints the background, the visible rows (with hover/selection
// highlights), and a scrollbar thumb when the content overflows.
func (l *List) Draw(canvas render.Canvas) {
	f := l.face()
	if f == nil {
		return
	}
	b := l.Bounds()
	canvas.FillRect(b, l.ColourOf(RoleSurface))

	rh := l.RowHeight()
	g := l.geomInfo()
	// Rows are as wide as the widest when scrolling across, so a highlight spans the
	// whole scrolled row; the clip keeps them off the gutter.
	rowW := g.viewW
	if l.hscroll {
		rowW = maxF(l.widest(), g.viewW)
	}
	ox := b.X - l.offsetX

	canvas.PushClip(geom.Rect{X: b.X, Y: b.Y, W: g.viewW, H: g.viewH})
	for i, it := range l.items {
		y := b.Y - l.offset + float64(i)*rh
		if y+rh < b.Y || y > b.Y+g.viewH {
			continue // not visible
		}
		row := geom.Rect{X: ox, Y: y, W: rowW, H: rh}
		switch {
		case i == l.selected:
			canvas.FillRect(row, l.ColourOf(RolePrimary))
		case i == l.hoverRow:
			canvas.FillRect(row, tint(l.ColourOf(RoleSurface), tintHover))
		}
		textColour := l.ColourOf(RoleText)
		if i == l.selected {
			textColour = l.ColourOf(RoleOnPrimary)
		}
		canvas.DrawText(it, geom.Point{X: ox + listRowPad, Y: vCenterY(f, y, rh)}, f, textColour)
	}
	canvas.PopClip()

	canvas.StrokeRect(b, l.ColourOf(RoleBorder), 1)

	if g.vOver {
		l.drawScrollbar(canvas, b)
	}
	if g.hOver {
		drawBar(canvas, l.scrollTrackH(), false, l.scrollMetricsH(), l.ColourOf(RoleBackground), l.ColourOf(RoleAccent))
		if g.vOver { // the corner where the two gutters meet
			canvas.FillRect(geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y + b.H - scrollbarWidth, W: scrollbarWidth, H: scrollbarWidth}, l.ColourOf(RoleBackground))
		}
	}
}

// scrollTrack is the scrollbar gutter along the right edge.
func (l *List) scrollTrack() geom.Rect {
	b := l.Bounds()
	return geom.Rect{X: b.X + b.W - scrollbarWidth, Y: b.Y, W: scrollbarWidth, H: l.geomInfo().viewH}
}

func (l *List) scrollMetrics() scrollMetrics {
	return scrollMetrics{content: l.ContentHeight(), view: l.geomInfo().viewH, offset: l.offset}
}

func (l *List) drawScrollbar(canvas render.Canvas, b geom.Rect) {
	drawBar(canvas, l.scrollTrack(), true, l.scrollMetrics(), l.ColourOf(RoleBackground), l.ColourOf(RoleAccent))
}

// HandleEvent handles hover tracking, clicking, wheel scrolling and keyboard
// navigation.
func (l *List) HandleEvent(ev *Event) bool {
	if !l.Enabled() {
		return false
	}
	g := l.geomInfo()
	if off, changed, ok := l.bar.handle(ev, l.scrollTrack(), g.vOver, true, l.scrollMetrics()); ok {
		if changed {
			l.offset = off
			l.clamp()
		}
		return true
	}
	if off, changed, ok := l.barH.handle(ev, l.scrollTrackH(), g.hOver, false, l.scrollMetricsH()); ok {
		if changed {
			l.offsetX = off
			l.clamp()
		}
		return true
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
	case EventDoubleClick:
		if i := l.rowAt(ev.Pos.Y); i >= 0 && ev.Button == render.MouseLeft {
			l.SetSelected(i)
			if l.onActivate != nil {
				l.onActivate(i)
			}
		}
		return true
	case EventWheel:
		// A sideways wheel, or Shift with the ordinary one, scrolls across when there
		// is somewhere to go.
		if dx := wheelAcross(ev); dx != 0 && l.maxOffsetX() > 0 {
			l.offsetX -= dx * wheelStep
			l.clamp()
			return true
		}
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
	case render.KeyLeft, render.KeyRight:
		if !l.hscroll {
			return false
		}
		if k == render.KeyLeft {
			l.offsetX -= wheelStep
		} else {
			l.offsetX += wheelStep
		}
		l.clamp()
	case render.KeyEnter:
		if l.selected >= 0 && l.onSelect != nil {
			l.onSelect(l.selected)
		}
		if l.selected >= 0 && l.onActivate != nil {
			l.onActivate(l.selected)
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
