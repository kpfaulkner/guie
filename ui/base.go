package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// BaseWidget provides tree wiring, state storage and default method
// implementations shared by all widgets. Embed it in a concrete widget and
// override Draw, Layout and MinSize (and, later, event handling) as needed.
type BaseWidget struct {
	bounds  geom.Rect
	visible bool
	enabled bool
	parent  Widget
	ctx     *treeContext
}

// NewBase returns a BaseWidget that is visible and enabled. Concrete widgets
// embed the result, e.g. &MyWidget{BaseWidget: ui.NewBase()}.
func NewBase() BaseWidget {
	return BaseWidget{visible: true, enabled: true}
}

// Parent returns the widget's parent, or nil for the root.
func (b *BaseWidget) Parent() Widget { return b.parent }

// Children returns nil; leaf widgets have no children.
func (b *BaseWidget) Children() []Widget { return nil }

// Bounds returns the widget's absolute rectangle.
func (b *BaseWidget) Bounds() geom.Rect { return b.bounds }

// SetBounds assigns the widget's absolute rectangle.
func (b *BaseWidget) SetBounds(r geom.Rect) { b.bounds = r }

// MinSize returns the zero size by default.
func (b *BaseWidget) MinSize() geom.Size { return geom.Size{} }

// Layout is a no-op by default.
func (b *BaseWidget) Layout() {}

// Draw is a no-op by default.
func (b *BaseWidget) Draw(c render.Canvas) {}

// Visible reports whether the widget is visible.
func (b *BaseWidget) Visible() bool { return b.visible }

// Enabled reports whether the widget is enabled.
func (b *BaseWidget) Enabled() bool { return b.enabled }

// SetVisible sets visibility and requests a re-layout.
func (b *BaseWidget) SetVisible(v bool) {
	b.visible = v
	b.Invalidate()
}

// SetEnabled sets the enabled state.
func (b *BaseWidget) SetEnabled(v bool) { b.enabled = v }

// Invalidate requests that the framework re-run layout before the next frame.
// It is safe to call before the widget is mounted (it simply does nothing).
func (b *BaseWidget) Invalidate() {
	if b.ctx != nil {
		b.ctx.markNeedsLayout()
	}
}

func (b *BaseWidget) mount(parent Widget, ctx *treeContext) {
	b.parent = parent
	b.ctx = ctx
}
