package ui

import "github.com/hajimehoshi/ebiten/v2"

// Widget is the interface implemented by every element in the UI tree.
//
// Coordinates are relative to the parent: a widget's Bounds are expressed in
// the coordinate space of its container. During drawing and event dispatch the
// parent passes an origin, which is the absolute screen position of the
// container's content area. A widget's absolute rectangle is therefore
// Bounds().Add(origin).
type Widget interface {
	// Bounds returns the widget's rectangle relative to its parent.
	Bounds() Rect
	// SetBounds updates the widget's rectangle.
	SetBounds(r Rect)

	// IsVisible reports whether the widget should be drawn and receive events.
	IsVisible() bool

	// Update advances any per-frame state. It is called once per frame.
	Update() error

	// Draw renders the widget onto dst. origin is the absolute screen position
	// of the parent's content area.
	Draw(dst *ebiten.Image, origin Point)

	// HandleEvent dispatches an input event to the widget. origin is the
	// absolute screen position of the parent's content area. It returns true if
	// the event was consumed and should not propagate further.
	HandleEvent(ev *Event, origin Point) bool
}

// BaseWidget provides default implementations for the Widget interface and is
// intended to be embedded by concrete widgets. Embedders typically override
// Draw and HandleEvent, and may override Update.
type BaseWidget struct {
	bounds  Rect
	visible bool
}

// NewBase returns a BaseWidget with the given bounds, initially visible.
func NewBase(r Rect) BaseWidget {
	return BaseWidget{bounds: r, visible: true}
}

// Bounds returns the widget's rectangle relative to its parent.
func (b *BaseWidget) Bounds() Rect { return b.bounds }

// SetBounds updates the widget's rectangle.
func (b *BaseWidget) SetBounds(r Rect) { b.bounds = r }

// IsVisible reports whether the widget is visible.
func (b *BaseWidget) IsVisible() bool { return b.visible }

// SetVisible sets the widget's visibility.
func (b *BaseWidget) SetVisible(v bool) { b.visible = v }

// Update is a no-op by default.
func (b *BaseWidget) Update() error { return nil }

// Draw is a no-op by default.
func (b *BaseWidget) Draw(dst *ebiten.Image, origin Point) {}

// HandleEvent ignores the event by default.
func (b *BaseWidget) HandleEvent(ev *Event, origin Point) bool { return false }
