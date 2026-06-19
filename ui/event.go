package ui

import "github.com/hajimehoshi/ebiten/v2"

// EventType enumerates the kinds of input events the framework dispatches.
type EventType int

const (
	// MouseMove is emitted when the cursor position changes.
	MouseMove EventType = iota
	// MouseDown is emitted when a mouse button is pressed.
	MouseDown
	// MouseUp is emitted when a mouse button is released.
	MouseUp
	// MouseWheel is emitted when the scroll wheel moves.
	MouseWheel
	// KeyDown is emitted when a key is pressed.
	KeyDown
	// KeyUp is emitted when a key is released.
	KeyUp
)

// Event describes a single input event. Pos is always the current cursor
// position in absolute screen coordinates, regardless of event type.
type Event struct {
	Type EventType

	// Pos is the cursor position in absolute screen coordinates.
	Pos Point

	// Button is the relevant mouse button for MouseDown / MouseUp events.
	Button ebiten.MouseButton

	// WheelX and WheelY are the scroll deltas for MouseWheel events.
	WheelX, WheelY float64

	// Key is the relevant key for KeyDown / KeyUp events.
	Key ebiten.Key
}
