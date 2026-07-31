package ui

import (
	"io/fs"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// EventType identifies the kind of input event delivered to a widget.
type EventType int

const (
	// EventPointerEnter is sent when the cursor moves onto a widget.
	EventPointerEnter EventType = iota
	// EventPointerLeave is sent when the cursor moves off a widget.
	EventPointerLeave
	// EventPointerMove is sent each frame to the widget capturing the pointer
	// (the press target), enabling drag interactions.
	EventPointerMove
	// EventPointerDown is sent when a mouse button is pressed over a widget.
	EventPointerDown
	// EventPointerUp is sent to the press target (pointer capture) when a mouse
	// button is released, even if the cursor has moved off it.
	EventPointerUp
	// EventClick is sent when a press and release land on the same widget.
	EventClick
	// EventWheel is sent to the widget under the cursor when the wheel moves.
	EventWheel
	// EventKeyDown is sent to the focused widget when a key is pressed.
	EventKeyDown
	// EventKeyUp is sent to the focused widget when a key is released.
	EventKeyUp
	// EventText is sent to the focused widget for each typed rune.
	EventText
	// EventFocusGained is sent to a widget when it becomes focused.
	EventFocusGained
	// EventFocusLost is sent to a widget when it loses focus.
	EventFocusLost
	// EventComposition is sent to the focused widget when the IME preedit
	// (uncommitted composition) changes. The preedit is carried in Event.Comp; an
	// empty Comp.Text ends/clears the composition. Committed text still arrives
	// separately via EventText.
	EventComposition
	// EventFileDrop is sent to the widget under the cursor when the user drops
	// files from outside the application onto the window, and bubbles from there.
	// The files are carried in Event.Files. Unlike in-process drag-and-drop there
	// is no enter/over/leave phase: the OS reports only the completed drop.
	EventFileDrop
)

// Event describes a single input event. Which fields are meaningful depends on
// Type: pointer events use Pos/Button, EventWheel uses Wheel, key events use
// Key, EventText uses Rune. Modifiers apply to all input events.
type Event struct {
	Type      EventType
	Pos       geom.Point // absolute cursor position
	Button    render.MouseButton
	Wheel     geom.Point // wheel delta for EventWheel
	Key       render.Key // key for EventKeyDown/EventKeyUp
	Rune      rune       // typed rune for EventText
	Modifiers render.ModifierSet
	Comp      render.Composition // IME preedit for EventComposition
	Files     FileDrop           // dropped files for EventFileDrop
}

// FileDrop is the payload of an OS file drop: the files the user dragged in from
// outside the application. It is delivered by EventFileDrop and to OnFileDrop
// handlers.
//
// Content is reached through FS; Names are its root entries, read once by the App
// so that every widget in the bubble chain does not walk the filesystem again.
// Names are base names — a dropped file has no path an application can keep, so
// anything needing to re-read the file later (reload, save-in-place) must ask the
// user for a real location instead. Names may include directories; use fs.Stat to
// tell them apart.
type FileDrop struct {
	FS    fs.FS
	Names []string
}

// ReadFile reads a dropped file's content by name. It is a convenience for the
// common case of wanting the bytes of one dropped file.
func (d FileDrop) ReadFile(name string) ([]byte, error) {
	if d.FS == nil {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(d.FS, name)
}

// hitTest returns the top-most visible widget whose bounds contain pos, going
// depth-first and preferring later (on-top) children. It returns nil if pos is
// outside w.
func hitTest(w Widget, pos geom.Point) Widget {
	if w == nil || !w.Visible() || !w.Bounds().Contains(pos) {
		return nil
	}
	children := w.Children()
	for i := len(children) - 1; i >= 0; i-- {
		if hit := hitTest(children[i], pos); hit != nil {
			return hit
		}
	}
	return w
}

// appendFocusables collects the visible, focusable widgets under w in tree
// (tab) order.
func appendFocusables(w Widget, out []Widget) []Widget {
	if w == nil || !w.Visible() {
		return out
	}
	if w.Focusable() {
		out = append(out, w)
	}
	for _, ch := range w.Children() {
		out = appendFocusables(ch, out)
	}
	return out
}
