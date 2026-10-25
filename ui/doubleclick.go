package ui

import (
	"time"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

const (
	// doubleClickInterval is the longest gap between two clicks that still makes
	// a double-click.
	doubleClickInterval = 400 * time.Millisecond
	// doubleClickSlop is the furthest (logical px, per axis) the pointer may move
	// between the two clicks.
	doubleClickSlop = 5
)

// clickRecord is the previous derived click, kept to recognise the next one as
// the second half of a double-click.
type clickRecord struct {
	target Widget
	pos    geom.Point
	at     time.Time
	button render.MouseButton
	valid  bool
}

// OnDoubleClick registers a handler for double-clicks on this widget. pos is the
// cursor position of the second click. Return true to consume the event;
// returning false lets it bubble to the ancestors. The two clicks that make up
// the double-click have already been delivered as ordinary EventClick events.
func (b *BaseWidget) OnDoubleClick(fn func(pos geom.Point) bool) { b.onDoubleClick = fn }

// doubleClickFn exposes the handler to the App for any widget (all embed
// BaseWidget), including ones that override HandleEvent.
func (b *BaseWidget) doubleClickFn() func(geom.Point) bool { return b.onDoubleClick }

// noteClick records a derived click on target and, if it completes a
// double-click, dispatches EventDoubleClick. Clicks that end a drag never reach
// here, so a drag is not counted.
func (a *App) noteClick(target Widget, pos geom.Point, btn render.MouseButton, mods render.ModifierSet) {
	now := a.now()
	prev := a.lastClick
	if !prev.valid || prev.target != target || prev.button != btn ||
		now.Sub(prev.at) > doubleClickInterval ||
		absF(pos.X-prev.pos.X) > doubleClickSlop || absF(pos.Y-prev.pos.Y) > doubleClickSlop {
		a.lastClick = clickRecord{target: target, pos: pos, at: now, button: btn, valid: true}
		return
	}
	a.lastClick = clickRecord{} // a third click begins a fresh sequence
	ev := Event{Type: EventDoubleClick, Pos: pos, Button: btn, Modifiers: mods}
	// Bubble like dispatch, but also consult OnDoubleClick at each level so
	// widgets that override HandleEvent do not shadow their own registered handler.
	for w := target; w != nil; w = w.Parent() {
		if w.HandleEvent(&ev) {
			break
		}
		if h, ok := w.(interface{ doubleClickFn() func(geom.Point) bool }); ok {
			if fn := h.doubleClickFn(); fn != nil && fn(pos) {
				break
			}
		}
	}
	a.bus.publish(ev)
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
