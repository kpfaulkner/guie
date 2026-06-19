package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const (
	menuTitlePad = 12
	menuRowPad   = 6
)

// MenuItem is one entry in a menu: a label and the action to run when chosen.
type MenuItem struct {
	Label  string
	Action func()
}

// NewMenuItem is a convenience constructor for a MenuItem.
func NewMenuItem(label string, action func()) MenuItem {
	return MenuItem{Label: label, Action: action}
}

// MenuBar is a horizontal bar of menu titles. Clicking a title opens a popup of
// that menu's items below it; once a menu is open, moving the cursor to another
// title switches to it. Choosing an item runs its action and closes the menu.
type MenuBar struct {
	BaseWidget
	titles  []string
	menus   [][]MenuItem
	hover   int // hovered title index, or -1
	openIdx int // open title index, or -1
	popup   *Popup
	font    render.FontFace
}

// NewMenuBar returns an empty MenuBar.
func NewMenuBar() *MenuBar {
	return &MenuBar{BaseWidget: NewBase(), hover: -1, openIdx: -1}
}

// AddMenu appends a menu with the given title and items.
func (m *MenuBar) AddMenu(title string, items ...MenuItem) {
	m.titles = append(m.titles, title)
	m.menus = append(m.menus, items)
}

func (m *MenuBar) face() render.FontFace {
	if m.font != nil {
		return m.font
	}
	return m.appTheme().Font
}

// SetFont overrides the menu bar's font face (nil falls back to the theme font).
func (m *MenuBar) SetFont(f render.FontFace) {
	m.font = f
	m.Invalidate()
}

// MinSize returns the total title width and one row tall.
func (m *MenuBar) MinSize() geom.Size {
	f := m.face()
	if f == nil {
		return geom.Size{}
	}
	var w float64
	for _, t := range m.titles {
		w += f.Measure(t).W + 2*menuTitlePad
	}
	return geom.Size{W: w, H: f.Measure("Ag").H + 2*menuRowPad}
}

// titleRect returns the rectangle of title i within the bar.
func (m *MenuBar) titleRect(i int) geom.Rect {
	f := m.face()
	b := m.Bounds()
	x := b.X
	for j := 0; j < i; j++ {
		x += f.Measure(m.titles[j]).W + 2*menuTitlePad
	}
	return geom.Rect{X: x, Y: b.Y, W: f.Measure(m.titles[i]).W + 2*menuTitlePad, H: b.H}
}

// titleAt returns the title index at absolute point p, or -1.
func (m *MenuBar) titleAt(p geom.Point) int {
	if !m.Bounds().Contains(p) {
		return -1
	}
	for i := range m.titles {
		if m.titleRect(i).Contains(p) {
			return i
		}
	}
	return -1
}

// Draw paints the bar and its titles, highlighting the hovered/open one.
func (m *MenuBar) Draw(canvas render.Canvas) {
	pal := m.appTheme().Palette
	f := m.face()
	if f == nil {
		return
	}
	canvas.FillRect(m.Bounds(), pal.Surface)
	for i, t := range m.titles {
		r := m.titleRect(i)
		switch {
		case i == m.openIdx:
			canvas.FillRect(r, pal.Primary)
		case i == m.hover:
			canvas.FillRect(r, lighten(pal.Surface, 1.25))
		}
		ts := f.Measure(t)
		canvas.DrawText(t, geom.Point{X: r.X + menuTitlePad, Y: r.Y + (r.H-ts.H)/2}, f, pal.Text)
	}
}

// HandleEvent opens/switches menus on hover and click.
func (m *MenuBar) HandleEvent(ev *Event) bool {
	switch ev.Type {
	case EventPointerMove:
		i := m.titleAt(ev.Pos)
		m.hover = i
		// While a menu is open, hovering a different title switches to it.
		if m.openIdx >= 0 && i >= 0 && i != m.openIdx {
			m.openMenu(i)
		}
		return true
	case EventPointerLeave:
		m.hover = -1
		return true
	case EventClick:
		if i := m.titleAt(ev.Pos); i >= 0 {
			m.openMenu(i)
			return true
		}
	}
	return false
}

// openMenu opens the popup for title i, replacing any currently open menu.
func (m *MenuBar) openMenu(i int) {
	if m.ctx == nil || i < 0 || i >= len(m.menus) {
		return
	}
	if m.popup != nil {
		m.ctx.close(m.popup)
	}

	f := m.face()
	rowH := f.Measure("Ag").H + 2*menuRowPad
	var w float64
	for _, it := range m.menus[i] {
		w = maxF(w, f.Measure(it.Label).W)
	}

	panel := NewContainer()
	panel.SetBackground(m.appTheme().Palette.Surface)
	panel.SetLayout(VBox(0))
	for _, it := range m.menus[i] {
		action := it.Action
		row := newMenuRow(it.Label, func() {
			if action != nil {
				action()
			}
			m.ctx.close(m.popup)
		})
		row.font = m.font
		panel.Add(row)
	}

	tr := m.titleRect(i)
	bounds := geom.Rect{X: tr.X, Y: tr.Y + tr.H, W: w + 2*menuRowPad, H: rowH * float64(len(m.menus[i]))}
	m.popup = NewPopup(panel, bounds, func() { m.openIdx = -1; m.popup = nil })
	m.ctx.open(m.popup)
	m.openIdx = i
}

// menuRow is a single clickable row inside a menu popup. Being its own widget,
// it gets per-row hover via enter/leave automatically.
type menuRow struct {
	BaseWidget
	label   string
	onClick func()
	hover   bool
	font    render.FontFace
}

func newMenuRow(label string, onClick func()) *menuRow {
	return &menuRow{BaseWidget: NewBase(), label: label, onClick: onClick}
}

func (r *menuRow) face() render.FontFace {
	if r.font != nil {
		return r.font
	}
	return r.appTheme().Font
}

func (r *menuRow) MinSize() geom.Size {
	f := r.face()
	if f == nil {
		return geom.Size{}
	}
	s := f.Measure(r.label)
	return geom.Size{W: s.W + 2*menuRowPad, H: s.H + 2*menuRowPad}
}

func (r *menuRow) Draw(canvas render.Canvas) {
	pal := r.appTheme().Palette
	f := r.face()
	if f == nil {
		return
	}
	b := r.Bounds()
	if r.hover {
		canvas.FillRect(b, pal.Accent)
	}
	s := f.Measure(r.label)
	canvas.DrawText(r.label, geom.Point{X: b.X + menuRowPad, Y: b.Y + (b.H-s.H)/2}, f, pal.Text)
}

func (r *menuRow) HandleEvent(ev *Event) bool {
	switch ev.Type {
	case EventPointerEnter:
		r.hover = true
		return true
	case EventPointerLeave:
		r.hover = false
		return true
	case EventClick:
		if r.onClick != nil {
			r.onClick()
		}
		return true
	}
	return false
}
