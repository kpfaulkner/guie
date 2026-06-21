package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	tabHPad = 12 // horizontal padding inside a tab title
	tabVPad = 6  // vertical padding of the tab strip
)

// tab is one pane: a title shown in the strip and its content widget.
type tab struct {
	title   string
	content Widget
}

// TabContainer shows a horizontal strip of tab titles above a content area.
// Clicking a title (or using Left/Right while focused) switches which pane is
// shown; only the active pane is drawn and receives events. All panes are
// mounted so they keep their state while hidden.
type TabContainer struct {
	BaseWidget
	tabs     []tab
	selected int
	hover    int
	focused  bool
	onChange func(int)
	font     render.FontFace
}

// NewTabContainer returns an empty TabContainer.
func NewTabContainer() *TabContainer {
	return &TabContainer{BaseWidget: NewBase(), hover: -1}
}

// OnChange registers the handler invoked with the newly selected tab index.
func (t *TabContainer) OnChange(fn func(int)) { t.onChange = fn }

// AddTab appends a pane with the given title and content widget.
func (t *TabContainer) AddTab(title string, content Widget) {
	t.tabs = append(t.tabs, tab{title: title, content: content})
	if t.ctx != nil {
		content.mount(content, t.self, t.ctx)
	}
	t.Invalidate()
}

// Selected returns the active tab index.
func (t *TabContainer) Selected() int { return t.selected }

// Select makes tab i active and fires OnTabChange if it changed.
func (t *TabContainer) Select(i int) {
	if i < 0 || i >= len(t.tabs) || i == t.selected {
		return
	}
	t.selected = i
	if t.onChange != nil {
		t.onChange(i)
	}
	t.Invalidate()
}

func (t *TabContainer) face() render.FontFace {
	if t.font != nil {
		return t.font
	}
	return t.appTheme().Font
}

// SetFont overrides the tab strip's font face (nil falls back to the theme font).
func (t *TabContainer) SetFont(f render.FontFace) {
	t.font = f
	t.Invalidate()
}

// Focusable reports whether the tabs can take focus (only when enabled).
func (t *TabContainer) Focusable() bool { return t.Enabled() }

func (t *TabContainer) stripHeight() float64 {
	f := t.face()
	if f == nil {
		return 0
	}
	return f.Measure("Ag").H + 2*tabVPad
}

// active returns the content widget of the selected tab, or nil.
func (t *TabContainer) active() Widget {
	if t.selected < 0 || t.selected >= len(t.tabs) {
		return nil
	}
	return t.tabs[t.selected].content
}

// Children returns only the active pane, so hit-testing and event bubbling are
// confined to the visible tab.
func (t *TabContainer) Children() []Widget {
	if a := t.active(); a != nil {
		return []Widget{a}
	}
	return nil
}

func (t *TabContainer) mount(self, parent Widget, ctx *treeContext) {
	t.BaseWidget.mount(self, parent, ctx)
	for _, tb := range t.tabs {
		tb.content.mount(tb.content, self, ctx)
	}
}

// titleRect returns the rectangle of tab title i within the strip.
func (t *TabContainer) titleRect(i int) geom.Rect {
	f := t.face()
	b := t.Bounds()
	x := b.X
	for j := 0; j < i; j++ {
		x += f.Measure(t.tabs[j].title).W + 2*tabHPad
	}
	return geom.Rect{X: x, Y: b.Y, W: f.Measure(t.tabs[i].title).W + 2*tabHPad, H: t.stripHeight()}
}

// contentRect returns the area below the strip available to the active pane.
func (t *TabContainer) contentRect() geom.Rect {
	b := t.Bounds()
	sh := t.stripHeight()
	return geom.Rect{X: b.X, Y: b.Y + sh, W: b.W, H: b.H - sh}
}

// titleAt returns the tab index at absolute point p, or -1.
func (t *TabContainer) titleAt(p geom.Point) int {
	b := t.Bounds()
	if p.Y < b.Y || p.Y > b.Y+t.stripHeight() {
		return -1
	}
	for i := range t.tabs {
		if t.titleRect(i).Contains(p) {
			return i
		}
	}
	return -1
}

// MinSize fits the widest content and the tab strip.
func (t *TabContainer) MinSize() geom.Size {
	f := t.face()
	if f == nil {
		return geom.Size{}
	}
	var stripW, cw, ch float64
	for _, tb := range t.tabs {
		stripW += f.Measure(tb.title).W + 2*tabHPad
		m := tb.content.MinSize()
		cw = maxF(cw, m.W)
		ch = maxF(ch, m.H)
	}
	return geom.Size{W: maxF(stripW, cw), H: t.stripHeight() + ch}
}

// Layout positions the active pane in the content area.
func (t *TabContainer) Layout() {
	if a := t.active(); a != nil {
		a.SetBounds(t.contentRect())
		a.Layout()
	}
}

// Draw paints the tab strip and the active pane.
func (t *TabContainer) Draw(canvas render.Canvas) {
	f := t.face()
	if f == nil {
		return
	}
	b := t.Bounds()
	sh := t.stripHeight()

	canvas.FillRect(geom.Rect{X: b.X, Y: b.Y, W: b.W, H: sh}, t.ColourOf(RoleSurface))
	for i, tb := range t.tabs {
		r := t.titleRect(i)
		switch {
		case i == t.selected:
			canvas.FillRect(r, t.ColourOf(RolePrimary))
		case i == t.hover:
			canvas.FillRect(r, lighten(t.ColourOf(RoleSurface), 1.25))
		}
		col := t.ColourOf(RoleText)
		if i == t.selected {
			col = t.ColourOf(RoleOnPrimary)
		}
		canvas.DrawText(tb.title, geom.Point{X: r.X + tabHPad, Y: vCenterY(f, r.Y, r.H)}, f, col)
	}
	// Separator line under the strip.
	canvas.DrawLine(geom.Point{X: b.X, Y: b.Y + sh}, geom.Point{X: b.X + b.W, Y: b.Y + sh}, t.ColourOf(RoleBorder), 1)

	if a := t.active(); a != nil && a.Visible() {
		cr := t.contentRect()
		canvas.PushClip(cr)
		a.Draw(canvas)
		canvas.PopClip()
	}

	if t.focused {
		canvas.StrokeRect(b, t.ColourOf(RoleAccent), 1)
	}
}

// HandleEvent switches tabs on click/Left/Right and tracks hover/focus.
func (t *TabContainer) HandleEvent(ev *Event) bool {
	if !t.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerMove:
		t.hover = t.titleAt(ev.Pos)
		return true
	case EventPointerLeave:
		t.hover = -1
		return true
	case EventClick:
		if i := t.titleAt(ev.Pos); i >= 0 {
			t.Select(i)
			return true
		}
	case EventFocusGained:
		t.focused = true
		return true
	case EventFocusLost:
		t.focused = false
		return true
	case EventKeyDown:
		switch ev.Key {
		case render.KeyLeft:
			t.Select(t.selected - 1)
			return true
		case render.KeyRight:
			t.Select(t.selected + 1)
			return true
		}
	}
	return false
}
