package guitest

import (
	"io/fs"
	"testing/fstest"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
	"github.com/kpfaulkner/guie/ui"
)

// Harness drives an App through the headless backend, one frame at a time. It
// accumulates synthesized input (mouse, keyboard) and applies it on the next
// Step, which runs the framework's Update then Draw and returns the frame's
// Recording. Low-level methods (MoveMouse, PressMouse, PressKey, …) build up an
// input frame; high-level helpers (Click, Drag, TypeKey, …) perform their own
// Steps. Harness is for tests and is not safe for concurrent use.
type Harness struct {
	// App is the application under test. Use App.SetContent (or the Harness
	// SetContent shortcut) to install the widget tree, and reach into it to read
	// widget state in assertions.
	App *ui.App
	// Font is the deterministic headless font installed as the app default.
	Font render.FontFace

	driver *driver
	last   *Recording
	err    error

	mouse geom.Point
	down  render.ButtonSet
	mods  render.ModifierSet

	// composition is the current IME preedit. Unlike the edge fields below it
	// persists across frames (it is a level), so the preedit stays visible until
	// changed, committed or cancelled.
	composition render.Composition

	// edges accumulated for the next Step, cleared after it
	pressed      render.ButtonSet
	released     render.ButtonSet
	wheel        geom.Point
	keysDown     []render.Key
	keysPressed  []render.Key
	keysReleased []render.Key
	runes        []rune
	dropped      fs.FS // files for the next Step, from DropFiles
}

// New builds an App wired to the headless backend at the given logical size and
// returns a Harness to drive it. Extra ui.AppOptions are applied after the
// headless defaults (driver, size, deterministic font), so a caller may override
// them — e.g. pass ui.WithFont to use a real font face instead.
func New(width, height int, opts ...ui.AppOption) *Harness {
	d := newDriver()
	f := NewFont(theme.Default().FontSize)
	base := []ui.AppOption{
		ui.WithDriver(d),
		ui.WithSize(width, height),
		ui.WithFont(f),
	}
	app := ui.NewApp(append(base, opts...)...)
	h := &Harness{App: app, Font: f, driver: d}
	// Run captures the hooks and fires the initial Resize, then returns
	// immediately (the headless driver does not loop).
	h.err = app.Run()
	return h
}

// SetContent installs w as the root of the widget tree (shortcut for
// App.SetContent).
func (h *Harness) SetContent(w ui.Widget) { h.App.SetContent(w) }

// DropFiles simulates the user dropping files from outside the application onto
// the window at the current mouse position, and runs one frame to deliver them.
// The map is name → content; names are base names, as a real drop provides.
//
// It mirrors the backend contract: the files are visible for exactly the frame
// they arrive, so a later Step sees no drop.
func (h *Harness) DropFiles(files map[string][]byte) *Recording {
	mapped := make(fstest.MapFS, len(files))
	for name, data := range files {
		mapped[name] = &fstest.MapFile{Data: data}
	}
	h.dropped = mapped
	return h.Step()
}

// Step runs one frame with the currently accumulated input, stores and returns
// the frame's Recording, and clears the per-frame input edges (held mouse
// buttons and keys persist).
func (h *Harness) Step() *Recording {
	in := render.InputState{
		MousePos:      h.mouse,
		MouseDown:     h.down,
		MousePressed:  h.pressed,
		MouseReleased: h.released,
		WheelDelta:    h.wheel,
		KeysDown:      h.keysDown,
		KeysPressed:   h.keysPressed,
		KeysReleased:  h.keysReleased,
		Runes:         h.runes,
		Modifiers:     h.mods,
		Composition:   h.composition,
		DroppedFiles:  h.dropped,
	}
	rec, err := h.driver.step(in)
	if err != nil {
		h.err = err
	}
	h.last = rec

	h.pressed, h.released = 0, 0
	h.wheel = geom.Point{}
	h.keysPressed, h.keysReleased, h.runes = nil, nil, nil
	h.dropped = nil // a drop is an edge: one frame only
	return rec
}

// Frame returns the most recently drawn Recording (nil before the first Step).
func (h *Harness) Frame() *Recording { return h.last }

// IMEEnabled reports whether the app last asked the backend to enable IME (true
// while an editable widget holds focus).
func (h *Harness) IMEEnabled() bool { return h.driver.imeEnabled }

// IMERect returns the caret rectangle the app last reported for IME
// candidate-window placement.
func (h *Harness) IMERect() geom.Rect { return h.driver.imeRect }

// Err returns the first error returned by the framework's Update hook, if any.
func (h *Harness) Err() error { return h.err }

// RequestClose simulates the user asking the OS to close the window and reports
// whether the close was allowed. It mirrors what a real driver does with
// render.Hooks.CloseRequested: with no handler installed (App.OnCloseRequest) the
// close always proceeds, so this returns true. A false result means the app
// vetoed the close; the loop is unaffected either way, so Step still works after
// one.
func (h *Harness) RequestClose() bool { return h.driver.requestClose() }

// CloseHandled reports whether the app has asked the backend to stop closing the
// window by itself — true while a close handler is installed via
// App.OnCloseRequest.
func (h *Harness) CloseHandled() bool { return h.driver.closeHandled }

// Resize reports a new logical surface size to the app (re-layout happens on the
// next Step).
func (h *Harness) Resize(width, height int) *Harness {
	h.driver.resize(width, height)
	return h
}

// --- low-level input (build the next frame; chainable) ----------------------

// MoveMouse sets the cursor position for subsequent frames.
func (h *Harness) MoveMouse(x, y float64) *Harness {
	h.mouse = geom.Point{X: x, Y: y}
	return h
}

// PressMouse marks a button as going down this frame (and held thereafter).
func (h *Harness) PressMouse(b render.MouseButton) *Harness {
	h.pressed = h.pressed.Set(b)
	h.down = h.down.Set(b)
	return h
}

// ReleaseMouse marks a button as going up this frame and clears its held state.
func (h *Harness) ReleaseMouse(b render.MouseButton) *Harness {
	h.released = h.released.Set(b)
	h.down = render.ButtonSet(uint8(h.down) &^ (1 << uint(b)))
	return h
}

// ScrollBy adds wheel movement for the next frame.
func (h *Harness) ScrollBy(dx, dy float64) *Harness {
	h.wheel.X += dx
	h.wheel.Y += dy
	return h
}

// SetModifiers sets the active keyboard modifiers for subsequent frames.
func (h *Harness) SetModifiers(m render.ModifierSet) *Harness {
	h.mods = m
	return h
}

// PressKey marks a key as going down this frame (and held thereafter).
func (h *Harness) PressKey(k render.Key) *Harness {
	h.keysPressed = append(h.keysPressed, k)
	h.keysDown = appendUnique(h.keysDown, k)
	return h
}

// ReleaseKey marks a key as going up this frame and clears its held state.
func (h *Harness) ReleaseKey(k render.Key) *Harness {
	h.keysReleased = append(h.keysReleased, k)
	h.keysDown = removeKey(h.keysDown, k)
	return h
}

// TypeRune queues a typed character (text input) for the next frame.
func (h *Harness) TypeRune(r rune) *Harness {
	h.runes = append(h.runes, r)
	return h
}

// TypeText queues a string of typed characters for the next frame.
func (h *Harness) TypeText(s string) *Harness {
	h.runes = append(h.runes, []rune(s)...)
	return h
}

// --- high-level gestures (perform their own Steps) --------------------------

// Click performs a full left click at (x,y): a press frame followed by a release
// frame. It returns the recording after the release.
func (h *Harness) Click(x, y float64) *Recording {
	h.MoveMouse(x, y).PressMouse(render.MouseLeft).Step()
	return h.ReleaseMouse(render.MouseLeft).Step()
}

// RightClick performs a full right click at (x,y), used to open context menus.
func (h *Harness) RightClick(x, y float64) *Recording {
	h.MoveMouse(x, y).PressMouse(render.MouseRight).Step()
	return h.ReleaseMouse(render.MouseRight).Step()
}

// Drag presses at (fromX,fromY), moves through a midpoint to (toX,toY) over
// several frames, then releases — enough movement to cross the drag threshold.
// It returns the recording after the release.
func (h *Harness) Drag(fromX, fromY, toX, toY float64) *Recording {
	h.MoveMouse(fromX, fromY).PressMouse(render.MouseLeft).Step()
	h.MoveMouse((fromX+toX)/2, (fromY+toY)/2).Step()
	h.MoveMouse(toX, toY).Step()
	return h.ReleaseMouse(render.MouseLeft).Step()
}

// TypeKey presses and releases a key over two frames (Tab, Enter, arrows, …) and
// returns the recording after the release.
func (h *Harness) TypeKey(k render.Key) *Recording {
	h.PressKey(k).Step()
	return h.ReleaseKey(k).Step()
}

// --- IME (input method editor) ---

// Compose sets the IME preedit (uncommitted composition) text with the caret at
// the given rune position and steps a frame. The preedit persists across frames
// until changed by another Compose, ended by CommitText, or CancelComposition.
// This simulates an IME-capable backend regardless of the real one's support.
func (h *Harness) Compose(text string, caret int) *Recording {
	h.composition = render.Composition{Text: text, Caret: caret}
	return h.Step()
}

// CommitText ends the active composition and delivers s as committed text in the
// same frame (the order a real IME produces on accept), then steps.
func (h *Harness) CommitText(s string) *Recording {
	h.composition = render.Composition{}
	h.runes = append(h.runes, []rune(s)...)
	return h.Step()
}

// CancelComposition ends the active composition without committing any text.
func (h *Harness) CancelComposition() *Recording {
	h.composition = render.Composition{}
	return h.Step()
}

func appendUnique(ks []render.Key, k render.Key) []render.Key {
	for _, x := range ks {
		if x == k {
			return ks
		}
	}
	return append(ks, k)
}

func removeKey(ks []render.Key, k render.Key) []render.Key {
	out := ks[:0]
	for _, x := range ks {
		if x != k {
			out = append(out, x)
		}
	}
	return out
}
