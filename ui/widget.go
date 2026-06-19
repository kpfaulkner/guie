package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
	"github.com/kpfaulkner/uiframework/theme"
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

	// HandleEvent dispatches an input event to the widget. It returns true if
	// the event was consumed. Step 4 delivers pointer events directly to the
	// widget under the cursor; step 5 adds bubbling, focus and keyboard events.
	HandleEvent(ev *Event) bool

	// Visible reports whether the widget should be drawn and receive input.
	Visible() bool
	// Enabled reports whether the widget accepts interaction.
	Enabled() bool
	// Focusable reports whether the widget can receive keyboard focus. Such
	// widgets are visited by Tab traversal and focused on click.
	Focusable() bool

	// mount connects the widget to the tree. The parent calls it when the widget
	// is added to an already-mounted tree, and containers call it recursively on
	// their children. It is implemented by BaseWidget and unexported, so every
	// widget must embed BaseWidget.
	mount(parent Widget, ctx *treeContext)
}

// treeContext is state shared by every widget in a mounted tree. It carries the
// re-layout request, the active theme, focus and popup hooks so widgets can
// programmatically take focus or open/close overlay popups.
type treeContext struct {
	requestLayout func()
	requestFocus  func(Widget)
	openPopup     func(*Popup)
	closePopup    func(*Popup)
	theme         *theme.Theme
	clipboard     render.Clipboard
}

func (t *treeContext) focus(w Widget) {
	if t != nil && t.requestFocus != nil {
		t.requestFocus(w)
	}
}

func (t *treeContext) open(p *Popup) {
	if t != nil && t.openPopup != nil {
		t.openPopup(p)
	}
}

func (t *treeContext) close(p *Popup) {
	if t != nil && t.closePopup != nil {
		t.closePopup(p)
	}
}

func (t *treeContext) markNeedsLayout() {
	if t != nil && t.requestLayout != nil {
		t.requestLayout()
	}
}
