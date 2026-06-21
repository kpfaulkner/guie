// Package ui is the public API of the framework: the App, windows, widgets,
// layouts, events and styling helpers that applications use. Application code
// depends only on this package (and geom/render/theme for value types); it
// never imports EBiten. The App drives the main loop through a render.Driver,
// keeping the graphics backend an internal detail.
package ui

import (
	"sync"
	"sync/atomic"

	"github.com/kpfaulkner/guie/geom"
	ebitenbackend "github.com/kpfaulkner/guie/internal/ebiten"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
)

// App owns the main loop, the root of the widget tree and top-level
// configuration. Construct one with NewApp, give it content with SetContent,
// and start it with Run.
//
// Concurrency: an App and its widget tree are not safe for concurrent use. All
// methods must be called from the UI goroutine — that is, from within Run
// (event callbacks, Update). The only exceptions are Do and Quit, which are
// safe to call from any goroutine; use Do to marshal work onto the UI goroutine
// from a background goroutine.
type App struct {
	driver    render.Driver
	cfg       render.Config
	theme     theme.Theme
	clipboard render.Clipboard
	shadows   bool // draw drop shadows under overlays/tooltips

	mu      sync.Mutex  // guards pending
	pending []func()    // work queued via Do, run at the start of each frame
	quit    atomic.Bool // set by Quit to stop the loop

	root        Widget
	ctx         *treeContext
	needsLayout bool
	surfaceSize geom.Size
	bus         *EventBus

	overlays []*Popup // open popups, bottom-to-top

	// per-frame hooks and running animations (advanced each frame in update)
	frameCbs []func(dt float64)
	anims    []*Animation

	accels []accelerator // global keyboard shortcuts

	// pointer/focus dispatch state
	hovered     Widget     // widget currently under the cursor
	pressTarget Widget     // widget that received the active pointer-down (capture)
	focused     Widget     // widget with keyboard focus
	prevPointer geom.Point // cursor pos at the previous frame
	havePrev    bool       // whether prevPointer has been set

	drag *dragSession // in-flight drag-and-drop, nil when none

	// tooltip state (hover-delay timed in Update ticks)
	lastPointer  geom.Point
	tooltipTicks int
	tooltipText  string
	tooltipPos   geom.Point
}

// NewApp creates an App with default configuration, then applies opts.
func NewApp(opts ...AppOption) *App {
	a := &App{
		driver:  ebitenbackend.New(),
		theme:   theme.Default(),
		shadows: true,
		cfg: render.Config{
			Title:     "guie",
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
	if a.clipboard == nil {
		a.clipboard = &memClipboard{}
	}

	a.bus = newEventBus()
	a.ctx = &treeContext{
		requestLayout: func() { a.needsLayout = true },
		requestFocus:  a.setFocus,
		openPopup:     a.openPopup,
		closePopup:    a.closePopup,
		theme:         &a.theme,
		clipboard:     a.clipboard,
	}
	return a
}

// memClipboard is the default in-process Clipboard (no OS integration).
type memClipboard struct{ text string }

func (c *memClipboard) ReadText() string   { return c.text }
func (c *memClipboard) WriteText(s string) { c.text = s }

// Theme returns the app's active theme.
func (a *App) Theme() theme.Theme { return a.theme }

// SetFont changes the default text face at runtime and triggers a re-layout.
// Widgets that don't override their font pick up the change on the next frame.
// Build a face with ui.DefaultFont(size) or ui.LoadFont(...).
func (a *App) SetFont(f render.FontFace) {
	a.theme.Font = f
	a.needsLayout = true
}

// Events returns the global event bus for subscribing to events across the
// whole UI.
func (a *App) Events() *EventBus { return a.bus }

// SetShadows enables or disables overlay/tooltip drop shadows at runtime.
func (a *App) SetShadows(v bool) { a.shadows = v }

// SetContent installs w as the root of the widget tree. The root is sized to
// fill the surface. SetContent may be called before or after Run.
func (a *App) SetContent(w Widget) {
	a.root = w
	if w != nil {
		w.mount(w, nil, a.ctx)
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

// Do schedules fn to run on the UI goroutine at the start of the next frame.
// It is safe to call from any goroutine and is the supported way to update the
// UI from background work.
func (a *App) Do(fn func()) {
	if fn == nil {
		return
	}
	a.mu.Lock()
	a.pending = append(a.pending, fn)
	a.mu.Unlock()
}

// Quit requests a clean shutdown of the main loop; Run then returns nil. It is
// safe to call from any goroutine.
func (a *App) Quit() { a.quit.Store(true) }

// runPending drains and runs work queued via Do, on the UI goroutine.
func (a *App) runPending() {
	a.mu.Lock()
	q := a.pending
	a.pending = nil
	a.mu.Unlock()
	for _, fn := range q {
		fn()
	}
}

func (a *App) update(in render.InputState) error {
	a.runPending()
	if a.quit.Load() {
		return render.ErrTerminated
	}
	// Run frame callbacks and animations first, so any value changes they make
	// are laid out and drawn this same frame.
	a.advanceFrame(nominalFrameDelta)
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
	// While a modal is open, confine Tab traversal to the modal's content so
	// focus can't leak to the blocked background.
	scope := a.root
	if m := a.modalActive(); m != nil {
		scope = m.content
	}
	list := appendFocusables(scope, nil)
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

	// Determine the hit target and whether the pointer is inside a popup,
	// honouring modal popups (which block the background entirely).
	var hit Widget
	var inPopup bool
	if m := a.modalActive(); m != nil {
		hit = hitTest(m.content, pos)
		inPopup = hit != nil
	} else {
		hit, inPopup = a.overlayHit(pos)
		if !inPopup {
			hit = hitTest(a.root, pos)
		}
	}

	// A press (any button) outside the open popups is swallowed. For a non-modal
	// stack it also dismisses them; a modal stack stays open (the scrim absorbs
	// clicks).
	if in.MousePressed != 0 && len(a.overlays) > 0 && !inPopup {
		if a.modalActive() == nil {
			a.closeAll()
		}
		return
	}

	if hit != a.hovered {
		if a.hovered != nil {
			a.sendTo(a.hovered, Event{Type: EventPointerLeave, Pos: pos})
		}
		if hit != nil {
			a.sendTo(hit, Event{Type: EventPointerEnter, Pos: pos})
		}
		a.hovered = hit
	}

	a.updateTooltip(pos)

	// Report movement to the captured widget (for dragging) or, when nothing is
	// captured, to the hovered widget (so lists/menus can track the cursor row).
	// Only on actual movement: holding the pointer still must not flood widgets
	// with a redundant move event every frame (which, e.g., would make a drawing
	// widget append duplicate points 60×/sec while stationary).
	moved := !a.havePrev || pos != a.prevPointer
	// Advance any drag first; once it has actually started, it intercepts
	// movement so the normal PointerMove is not also delivered to the source
	// (e.g. a dragged slider must not also slide).
	if a.drag != nil && moved {
		a.advanceDrag(pos, hit)
	}
	if a.drag == nil || !a.drag.started {
		moveTarget := a.pressTarget
		if moveTarget == nil {
			moveTarget = a.hovered
		}
		if moved && moveTarget != nil {
			a.dispatch(moveTarget, Event{Type: EventPointerMove, Pos: pos, Modifiers: in.Modifiers})
		}
	}
	a.prevPointer = pos
	a.havePrev = true

	if (in.WheelDelta.X != 0 || in.WheelDelta.Y != 0) && hit != nil {
		a.dispatch(hit, Event{Type: EventWheel, Pos: pos, Wheel: in.WheelDelta, Modifiers: in.Modifiers})
	}

	// A right-press over a widget that has a context menu opens it at the cursor
	// instead of dispatching a normal press.
	if in.MousePressed.Has(render.MouseRight) && !inPopup {
		if t := contextTarget(hit); t != nil {
			a.hideTooltip()
			a.ShowContextMenu(pos, t.ContextMenu()...)
			return
		}
	}

	// Presses: the first button down captures the pointer (and moves focus);
	// further buttons are delivered to the same captured widget.
	for _, btn := range mouseButtons {
		if !in.MousePressed.Has(btn) {
			continue
		}
		if a.pressTarget == nil {
			a.hideTooltip()
			a.focusFromPointer(hit)
			a.pressTarget = hit
			// A left press on a drag source arms a pending drag; it only
			// becomes a real drag once movement crosses the threshold, so the
			// widget stays clickable.
			if btn == render.MouseLeft {
				if src := dragSourceOf(hit); src != nil {
					a.drag = &dragSession{source: src, origin: pos, last: pos}
				}
			}
		}
		if a.pressTarget != nil {
			a.dispatch(a.pressTarget, Event{Type: EventPointerDown, Pos: pos, Button: btn, Modifiers: in.Modifiers})
		}
	}

	// Releases: deliver to the captured widget, deriving a click when the release
	// lands within its bounds. Capture is held until all buttons are up.
	if a.pressTarget != nil {
		dragEnding := a.drag != nil && a.drag.started
		for _, btn := range mouseButtons {
			if !in.MouseReleased.Has(btn) {
				continue
			}
			a.dispatch(a.pressTarget, Event{Type: EventPointerUp, Pos: pos, Button: btn, Modifiers: in.Modifiers})
			// A release that ends a drag does not also derive a click.
			if !dragEnding && a.pressTarget.Bounds().Contains(pos) {
				a.dispatch(a.pressTarget, Event{Type: EventClick, Pos: pos, Button: btn, Modifiers: in.Modifiers})
			}
		}
		if in.MouseDown == 0 {
			a.pressTarget = nil
			if a.drag != nil {
				a.finishDrag(pos)
			}
		}
	}
}

// mouseButtons is the dispatch order for pointer buttons.
var mouseButtons = []render.MouseButton{render.MouseLeft, render.MouseMiddle, render.MouseRight}

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
		if k == render.KeyEscape && a.drag != nil {
			a.cancelDrag()
			continue
		}
		if k == render.KeyEscape && len(a.overlays) > 0 {
			a.closeTop()
			continue
		}
		if k == render.KeyTab {
			if in.Modifiers.Has(render.ModShift) {
				a.moveFocus(-1)
			} else {
				a.moveFocus(1)
			}
			continue
		}
		// Global accelerators take precedence over the focused widget.
		if a.runAccelerator(k, in.Modifiers) {
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
	a.drawOverlays(c)
	a.drawDrag(c)
	a.drawTooltip(c)
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
	a.layoutOverlays()
	a.needsLayout = false
}
