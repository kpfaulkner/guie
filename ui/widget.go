package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// Widget is the interface implemented by every node in the retained UI tree.
// Custom widgets satisfy it by embedding BaseWidget, which supplies the tree
// wiring and sensible defaults.
//
// Coordinates are absolute: a widget's Bounds are expressed in the surface's
// coordinate space, assigned by its parent during layout. Drawing therefore
// uses Bounds directly against the Canvas; no per-call origin is threaded
// through the tree.
type Widget interface {
	// Parent returns the widget's parent, or nil for the root.
	Parent() Widget
	// Children returns the widget's child widgets, or nil for a leaf.
	Children() []Widget

	// Bounds returns the widget's absolute rectangle.
	Bounds() geom.Rect
	// SetBounds assigns the widget's absolute rectangle.
	SetBounds(r geom.Rect)
	// MinSize returns the widget's intrinsic minimum size, used by layout.
	MinSize() geom.Size
	// Layout positions the widget's children within its Bounds. Leaf widgets
	// may leave this empty.
	Layout()

	// Draw paints the widget (and, for containers, its children) onto c.
	Draw(c render.Canvas)

	// Visible reports whether the widget should be drawn and receive input.
	Visible() bool
	// Enabled reports whether the widget accepts interaction.
	Enabled() bool

	// mount connects the widget to the tree. The parent calls it when the widget
	// is added to an already-mounted tree, and containers call it recursively on
	// their children. It is implemented by BaseWidget and unexported, so every
	// widget must embed BaseWidget.
	mount(parent Widget, ctx *treeContext)
}

// treeContext is state shared by every widget in a mounted tree. For now it
// carries only the re-layout request; it will grow as later steps add focus
// management and the event bus.
type treeContext struct {
	requestLayout func()
}

func (t *treeContext) markNeedsLayout() {
	if t != nil && t.requestLayout != nil {
		t.requestLayout()
	}
}
