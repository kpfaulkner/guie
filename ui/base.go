package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
	"github.com/kpfaulkner/guie/theme"
)

// BaseWidget provides tree wiring, state storage and default method
// implementations shared by all widgets. Embed it in a concrete widget and
// override Draw, Layout and MinSize (and, later, event handling) as needed.
type BaseWidget struct {
	bounds      geom.Rect
	visible     bool
	enabled     bool
	tooltip     string
	contextMenu []MenuItem                // shown on right-click, empty for none
	drag        *dragConfig               // drag-source / drop-target config, nil until opted in
	colors      map[ColorRole]color.Color // per-widget color overrides
	self        Widget                    // this widget's interface identity
	parent      Widget
	ctx         *treeContext
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

// HandleEvent ignores the event by default.
func (b *BaseWidget) HandleEvent(ev *Event) bool { return false }

// Visible reports whether the widget is visible.
func (b *BaseWidget) Visible() bool { return b.visible }

// Enabled reports whether the widget is enabled.
func (b *BaseWidget) Enabled() bool { return b.enabled }

// Focusable reports false by default; focusable widgets override it.
func (b *BaseWidget) Focusable() bool { return false }

// Tooltip returns the widget's hover hint text (empty by default).
func (b *BaseWidget) Tooltip() string { return b.tooltip }

// SetTooltip sets the hover hint text shown after the pointer rests on the
// widget. An empty string disables it.
func (b *BaseWidget) SetTooltip(s string) { b.tooltip = s }

// ContextMenu returns the items shown when the widget is right-clicked (nil for
// none).
func (b *BaseWidget) ContextMenu() []MenuItem { return b.contextMenu }

// SetContextMenu sets the items shown in a popup menu when the widget is
// right-clicked. Pass no items to clear it. The framework shows the menu at the
// cursor automatically; an item whose Action opens a dialog works because the
// menu closes before the action runs.
func (b *BaseWidget) SetContextMenu(items ...MenuItem) { b.contextMenu = items }

// RequestFocus asks the framework to give this widget keyboard focus. It is a
// no-op before the widget is mounted.
func (b *BaseWidget) RequestFocus() {
	if b.ctx != nil {
		b.ctx.focus(b.self)
	}
}

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

func (b *BaseWidget) mount(self, parent Widget, ctx *treeContext) {
	b.self = self
	b.parent = parent
	b.ctx = ctx
}

// appTheme returns the active theme, or a default if the widget is not yet
// mounted. Widgets read it during Draw and MinSize to resolve fonts and colors.
func (b *BaseWidget) appTheme() theme.Theme {
	if b.ctx != nil && b.ctx.theme != nil {
		return *b.ctx.theme
	}
	return theme.Default()
}

// clipboard returns the app clipboard, or nil if the widget is not yet mounted.
func (b *BaseWidget) clipboard() render.Clipboard {
	if b.ctx != nil {
		return b.ctx.clipboard
	}
	return nil
}

// cornerRadius returns the theme's default control corner radius.
func (b *BaseWidget) cornerRadius() float64 { return b.appTheme().CornerRadius }
