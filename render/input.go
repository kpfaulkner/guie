package render

import "github.com/kpfaulkner/guie/geom"

// MouseButton identifies a mouse button.
type MouseButton int

const (
	// MouseLeft is the primary (left) mouse button.
	MouseLeft MouseButton = iota
	// MouseMiddle is the middle mouse button.
	MouseMiddle
	// MouseRight is the secondary (right) mouse button.
	MouseRight
)

// ButtonSet is a bitset of mouse buttons.
type ButtonSet uint8

// Set returns s with b added.
func (s ButtonSet) Set(b MouseButton) ButtonSet { return s | (1 << b) }

// Has reports whether b is present in s.
func (s ButtonSet) Has(b MouseButton) bool { return s&(1<<b) != 0 }

// Modifier is a keyboard modifier flag.
type Modifier uint8

const (
	// ModShift is the Shift modifier.
	ModShift Modifier = 1 << iota
	// ModControl is the Control modifier.
	ModControl
	// ModAlt is the Alt/Option modifier.
	ModAlt
	// ModMeta is the Meta/Command/Windows modifier.
	ModMeta
	// ModPrimary is the platform's primary shortcut modifier: Command (Meta) on
	// macOS, Control on every other platform. Widgets test ModPrimary for
	// clipboard and selection shortcuts (copy/cut/paste/select-all) so they
	// follow the host platform's convention. The backend sets it alongside the
	// concrete ModControl/ModMeta bit it stands in for.
	ModPrimary
)

// ModifierSet is a bitset of active keyboard modifiers.
type ModifierSet uint8

// Has reports whether modifier m is active.
func (s ModifierSet) Has(m Modifier) bool { return uint8(s)&uint8(m) != 0 }

// Key is a backend-neutral keyboard key identifier.
type Key int

// Key constants. The set covers the common keys needed by widgets; backends map
// their native key codes onto these and report KeyUnknown for the rest.
const (
	KeyUnknown Key = iota
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyTab
	KeySpace
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyLeft
	KeyRight
	KeyUp
	KeyDown

	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ

	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
)

// InputState is the backend-neutral snapshot of input for a single frame. The
// "Pressed"/"Released" fields report edges (transitions during this frame),
// while the "Down"/"KeysDown" fields report the level (held state).
type InputState struct {
	// MousePos is the cursor position in logical pixels.
	MousePos geom.Point
	// MouseDown is the set of buttons currently held.
	MouseDown ButtonSet
	// MousePressed is the set of buttons that went down this frame.
	MousePressed ButtonSet
	// MouseReleased is the set of buttons that went up this frame.
	MouseReleased ButtonSet
	// WheelDelta is the scroll wheel movement this frame (x, y).
	WheelDelta geom.Point

	// KeysDown is the set of keys currently held.
	KeysDown []Key
	// KeysPressed is the set of keys that went down this frame.
	KeysPressed []Key
	// KeysReleased is the set of keys that went up this frame.
	KeysReleased []Key
	// Runes is the text input produced this frame, in order.
	Runes []rune
	// Modifiers is the set of modifiers active this frame.
	Modifiers ModifierSet
}
