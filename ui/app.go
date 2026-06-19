// Package ui is the public API of the framework: the App, windows, widgets,
// layouts, events and styling helpers that applications use. Application code
// depends only on this package (and geom/render/theme for value types); it
// never imports EBiten. The App drives the main loop through a render.Driver,
// keeping the graphics backend an internal detail.
package ui

import (
	ebitenbackend "github.com/kpfaulkner/uiframework/backend/ebiten"
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/theme"
)

// App owns the main loop, the root of the widget tree and top-level
// configuration. Construct one with NewApp, give it content with SetContent,
// and start it with Run.
type App struct {
	driver render.Driver
	cfg    render.Config
	theme  theme.Theme

	root        Widget
	ctx         *treeContext
	needsLayout bool
	surfaceSize geom.Size
	bus         *EventBus

	// pointer/focus dispatch state
	hovered     Widget // widget currently under the cursor
	pressTarget Widget // widget that received the active pointer-down (capture)
	focused     Widget // widget with keyboard focus
}

// NewApp creates an App with default configuration, then applies opts.
func NewApp(opts ...AppOption) *App {
	a := &App{
		driver: ebitenbackend.New(),
		theme:  theme.Default(),
		cfg: render.Config{
			Title:     "uiframework",
			Width:     800,
			Height:    600,
			Resizable: true,
		},
	}
	for _, o := range opts {
		o(a)
	}

	// Default the theme font and the clear color from the theme if unset.
	if a.theme.Font == nil {
		a.theme.Font = ebitenbackend.DefaultFont(a.theme.FontSize)
	}
	if a.cfg.Background == nil {
		a.cfg.Background = a.theme.Palette.Background
	}

	a.bus = newEventBus()
	a.ctx = &treeContext{
		requestLayout: func() { a.needsLayout = true },
		requestFocus:  a.setFocus,
		theme:         &a.theme,
	}
	return a
}

// Theme returns the app's active theme.
func (a *App) Theme() theme.Theme { return a.theme }

// Events returns the global event bus for subscribing to events across the
// whole UI.
func (a *App) Events() *EventBus { return a.bus }

// SetContent installs w as the root of the widget tree. The root is sized to
// fill the surface. SetContent may be called before or after Run.
func (a *App) SetContent(w Widget) {
	a.root = w
	if w != nil {
		w.mount(nil, a.ctx)
		if a.surfaceSize.W > 0 {
			w.SetBounds(geom.Rect{W: a.surfaceSize.W, H: a.surfaceSize.H})
		}
	}
	a.needsLayout = true
}

// Run starts the main loop. It blocks until the window is closed or an error
// occurs.
func (a *App) Run() error {
	return a.driver.Run(a.cfg, render.Hooks{
		Update: a.update,
		Draw:   a.draw,
		Resize: a.resize,
	})
}

func (a *App) update(in render.InputState) error {
	// Keep layout current before hit-testing, then dispatch input.
	a.layoutIfNeeded()
	a.dispatchPointer(in)
	a.dispatchKeyboard(in)
	return nil
}

// dispatch sends ev to target and bubbles it up through the ancestors until a
// widget consumes it (returns true). Every dispatched event is also published
// to the event bus. It returns whether the event was consumed.
func (a *App) dispatch(target Widget, ev Event) bool {
	consumed := false
	for w := target; w != nil; w = w.Parent() {
		if w.HandleEvent(&ev) {
			consumed = true
			break
		}
	}
	a.bus.publish(ev)
	return consumed
}

// sendTo delivers ev to a single widget without bubbling (used for targeted
// enter/leave/focus events) and publishes it to the bus.
func (a *App) sendTo(w Widget, ev Event) {
	if w != nil {
		w.HandleEvent(&ev)
	}
	a.bus.publish(ev)
}

// setFocus moves keyboard focus to w (nil clears focus), sending focus-lost and
// focus-gained events as appropriate.
func (a *App) setFocus(w Widget) {
	if w == a.focused {
		return
	}
	if a.focused != nil {
		a.sendTo(a.focused, Event{Type: EventFocusLost})
	}
	a.focused = w
	if w != nil {
		a.sendTo(w, Event{Type: EventFocusGained})
	}
}

// moveFocus advances focus to the next (delta=+1) or previous (delta=-1)
// focusable widget in tree order, wrapping around.
func (a *App) moveFocus(delta int) {
	list := appendFocusables(a.root, nil)
	if len(list) == 0 {
		a.setFocus(nil)
		return
	}
	idx := -1
	for i, w := range list {
		if w == a.focused {
			idx = i
			break
		}
	}
	var next int
	switch {
	case idx >= 0:
		next = (idx + delta + len(list)) % len(list)
	case delta < 0:
		next = len(list) - 1
	default:
		next = 0
	}
	a.setFocus(list[next])
}

// dispatchPointer translates the frame's mouse input into pointer events. Hover
// enter/leave follow the widget under the cursor; a press captures its target
// so the matching release goes to it, and a click is derived when the release
// lands on the same widget. Clicking also moves focus to the nearest focusable
// widget in the hit chain.
func (a *App) dispatchPointer(in render.InputState) {
	if a.root == nil {
		return
	}
	pos := in.MousePos
	hit := hitTest(a.root, pos)

	if hit != a.hovered {
		if a.hovered != nil {
			a.sendTo(a.hovered, Event{Type: EventPointerLeave, Pos: pos})
		}
		if hit != nil {
			a.sendTo(hit, Event{Type: EventPointerEnter, Pos: pos})
		}
		a.hovered = hit
	}

	if (in.WheelDelta.X != 0 || in.WheelDelta.Y != 0) && hit != nil {
		a.dispatch(hit, Event{Type: EventWheel, Pos: pos, Wheel: in.WheelDelta, Modifiers: in.Modifiers})
	}

	if in.MousePressed.Has(render.MouseLeft) {
		a.focusFromPointer(hit)
		if hit != nil {
			a.dispatch(hit, Event{Type: EventPointerDown, Pos: pos, Button: render.MouseLeft, Modifiers: in.Modifiers})
			a.pressTarget = hit
		} else {
			a.pressTarget = nil
		}
	}

	if in.MouseReleased.Has(render.MouseLeft) && a.pressTarget != nil {
		a.dispatch(a.pressTarget, Event{Type: EventPointerUp, Pos: pos, Button: render.MouseLeft, Modifiers: in.Modifiers})
		if a.pressTarget.Bounds().Contains(pos) {
			a.dispatch(a.pressTarget, Event{Type: EventClick, Pos: pos, Button: render.MouseLeft, Modifiers: in.Modifiers})
		}
		a.pressTarget = nil
	}
}

// focusFromPointer moves focus to the nearest focusable widget at or above hit,
// or clears focus if there is none (e.g. clicking empty space).
func (a *App) focusFromPointer(hit Widget) {
	for w := hit; w != nil; w = w.Parent() {
		if w.Focusable() {
			a.setFocus(w)
			return
		}
	}
	a.setFocus(nil)
}

// dispatchKeyboard routes key and text input to the focused widget. Tab and
// Shift-Tab move focus instead of being delivered.
func (a *App) dispatchKeyboard(in render.InputState) {
	for _, k := range in.KeysPressed {
		if k == render.KeyTab {
			if in.Modifiers.Has(render.ModShift) {
				a.moveFocus(-1)
			} else {
				a.moveFocus(1)
			}
			continue
		}
		if a.focused != nil {
			a.dispatch(a.focused, Event{Type: EventKeyDown, Key: k, Modifiers: in.Modifiers})
		}
	}
	for _, k := range in.KeysReleased {
		if a.focused != nil {
			a.dispatch(a.focused, Event{Type: EventKeyUp, Key: k, Modifiers: in.Modifiers})
		}
	}
	for _, r := range in.Runes {
		if a.focused != nil {
			a.dispatch(a.focused, Event{Type: EventText, Rune: r, Modifiers: in.Modifiers})
		}
	}
}

func (a *App) draw(c render.Canvas) {
	if a.root != nil && a.root.Visible() {
		a.root.Draw(c)
	}
}

func (a *App) resize(width, height int) {
	a.surfaceSize = geom.Size{W: float64(width), H: float64(height)}
	a.needsLayout = true
}

// layoutIfNeeded re-runs layout from the root when the tree has been marked
// dirty, sizing the root to the full surface first.
func (a *App) layoutIfNeeded() {
	if !a.needsLayout || a.root == nil {
		return
	}
	a.root.SetBounds(geom.Rect{W: a.surfaceSize.W, H: a.surfaceSize.H})
	a.root.Layout()
	a.needsLayout = false
}
