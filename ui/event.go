package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// EventType identifies the kind of input event delivered to a widget.
type EventType int

const (
	// EventPointerEnter is sent when the cursor moves onto a widget.
	EventPointerEnter EventType = iota
	// EventPointerLeave is sent when the cursor moves off a widget.
	EventPointerLeave
	// EventPointerDown is sent when a mouse button is pressed over a widget.
	EventPointerDown
	// EventPointerUp is sent when a mouse button is released. It is delivered to
	// the widget that received the matching EventPointerDown (pointer capture),
	// even if the cursor has since moved off it.
	EventPointerUp
)

// Event describes a single input event. Step 5 will extend this with keyboard,
// wheel and focus events and an event bus; for now it carries pointer state.
type Event struct {
	Type   EventType
	Pos    geom.Point // absolute cursor position
	Button render.MouseButton
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
