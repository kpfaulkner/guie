package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	tabHPad       = 12 // horizontal padding inside a tab title
	tabVPad       = 6  // vertical padding of the tab strip
	tabIndicatorH = 2  // thickness of the active-tab underline
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

	hoverFades []fade // per-tab hover level, parallel to tabs

	// Active-tab underline. It slides between tabs: while indAnim runs it is
	// drawn at the blend of indFrom and indTo (x relative to the strip's left
	// edge, so it survives a resize); once settled it snaps to the live rect.
	indFrom, indTo tabSpan
	indT           float64
	indAnim        *Animation
	onChange       func(int)
	font           render.FontFace
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
	t.hoverFades = append(t.hoverFades, fade{})
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
	t.slideIndicator(t.selected, i)
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

	surface := t.ColourOf(RoleSurface)
	canvas.FillRect(geom.Rect{X: b.X, Y: b.Y, W: b.W, H: sh}, surface)
	for i, tb := range t.tabs {
		r := t.titleRect(i)
		hv := t.hoverFades[i].level
		col := mixColour(t.ColourOf(RoleTextMuted), t.ColourOf(RoleText), hv)
		if i == t.selected {
			// The active tab is raised a shade and its label is full-strength;
			// the underline below marks it.
			canvas.FillRect(r, tint(surface, tintHover))
			col = t.ColourOf(RoleText)
		} else if hv > 0 {
			canvas.FillRect(r, withAlpha(tint(surface, tintHover), hv))
		}
		canvas.DrawText(tb.title, geom.Point{X: r.X + tabHPad, Y: vCenterY(f, r.Y, r.H)}, f, col)
	}
	// Separator line under the strip, with the active-tab underline over it.
	canvas.DrawLine(geom.Point{X: b.X, Y: b.Y + sh}, geom.Point{X: b.X + b.W, Y: b.Y + sh}, t.ColourOf(RoleBorder), 1)
	if len(t.tabs) > 0 {
		u := t.indicatorSpan()
		canvas.FillRect(geom.Rect{X: b.X + u.x, Y: b.Y + sh - tabIndicatorH, W: u.w, H: tabIndicatorH}, t.ColourOf(RoleAccent))
	}

	if a := t.active(); a != nil && a.Visible() {
		cr := t.contentRect()
		canvas.PushClip(cr)
		a.Draw(canvas)
		canvas.PopClip()
	}

	if t.focused {
		strokeFocusable(canvas, b, 0, t.ColourOf(RoleBorder), t.ColourOf(RoleAccent), true)
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
		t.syncHoverFades()
		return true
	case EventPointerLeave:
		t.hover = -1
		t.syncHoverFades()
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

// tabSpan is a horizontal extent within the strip: x from the strip's left edge.
type tabSpan struct{ x, w float64 }

// span returns tab i's extent relative to the strip's left edge.
func (t *TabContainer) span(i int) tabSpan {
	r := t.titleRect(i)
	return tabSpan{x: r.X - t.Bounds().X, w: r.W}
}

// syncHoverFades points each tab's hover fade at whether it is hovered.
func (t *TabContainer) syncHoverFades() {
	for i := range t.hoverFades {
		target := 0.0
		if i == t.hover {
			target = 1
		}
		t.fadeTo(&t.hoverFades[i], target)
	}
}

// slideIndicator starts the underline sliding from tab from to tab to. Before
// mount there is nothing to animate, so the underline just lands on the new tab.
func (t *TabContainer) slideIndicator(from, to int) {
	if t.indAnim != nil {
		t.indAnim.Stop()
		t.indAnim = nil
	}
	if t.ctx == nil || t.ctx.animate == nil || t.face() == nil ||
		from < 0 || from >= len(t.tabs) {
		t.indT = 1
		return
	}
	t.indFrom, t.indTo = t.indicatorSpan(), t.span(to)
	t.indT = 0
	t.indAnim = t.ctx.animate(transitionDuration*1.5, EaseOut, func(v float64) { t.indT = v })
	t.indAnim.OnDone(func() { t.indT = 1; t.indAnim = nil })
}

// indicatorSpan returns where the underline is drawn right now: the live extent
// of the selected tab once settled, or the in-flight blend while sliding.
func (t *TabContainer) indicatorSpan() tabSpan {
	if t.indAnim == nil || t.indT >= 1 {
		return t.span(t.selected)
	}
	return tabSpan{
		x: t.indFrom.x + (t.indTo.x-t.indFrom.x)*t.indT,
		w: t.indFrom.w + (t.indTo.w-t.indFrom.w)*t.indT,
	}
}
