package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// CanvasWidget is a widget whose appearance and input handling are supplied as
// callbacks. It is the escape hatch for charts, plots, drawing surfaces and
// game HUDs: anything that wants to paint with the render.Canvas primitives
// without defining a widget type.
//
// The draw callback receives the widget's absolute bounds and the same canvas
// the rest of the tree draws on, so coordinates are absolute. The canvas is
// clipped to the bounds around the call. Read colours with ColourOf so the
// canvas follows App.SetTheme.
//
// Draw runs only when a frame is presented, and presenting is on demand. State
// changed from an event callback needs nothing more; state changed elsewhere
// needs App.Invalidate (or App.Do from another goroutine). The widget's own
// Invalidate only requests a re-layout, not a frame. A continuously animated
// canvas should use App.Animate or App.OnFrame.
//
// Heavy canvases can render into a RenderTarget and DrawImage it from the draw
// callback; no caching is built in because only the app knows when it is stale.
//
// For anything with child widgets or its own layout, embed BaseWidget instead.
type CanvasWidget struct {
	BaseWidget
	draw      func(c render.Canvas, bounds geom.Rect)
	onEvent   func(ev *Event) bool
	minSize   geom.Size
	focusable bool

	hover   bool
	focused bool
}

// CanvasOption configures a CanvasWidget during NewCanvasWidget.
type CanvasOption func(*CanvasWidget)

// CanvasMinSize sets the minimum size reported to layout (default 0x0).
func CanvasMinSize(w, h float64) CanvasOption {
	return func(cw *CanvasWidget) { cw.minSize = geom.Size{W: w, H: h} }
}

// CanvasOnEvent sets the callback that receives every event routed to the
// widget. Return true to consume the event, false to let it bubble.
func CanvasOnEvent(fn func(ev *Event) bool) CanvasOption {
	return func(cw *CanvasWidget) { cw.onEvent = fn }
}

// CanvasFocusable makes the widget take keyboard focus: it joins Tab traversal,
// takes focus on click, receives key and text events, and shows the standard
// focus ring. Without it keyboard events never reach the widget.
func CanvasFocusable() CanvasOption {
	return func(cw *CanvasWidget) { cw.focusable = true }
}

// NewCanvasWidget returns a CanvasWidget painted by draw. A nil draw is allowed
// and draws nothing.
func NewCanvasWidget(draw func(c render.Canvas, bounds geom.Rect), opts ...CanvasOption) *CanvasWidget {
	cw := &CanvasWidget{BaseWidget: NewBase(), draw: draw}
	for _, o := range opts {
		o(cw)
	}
	return cw
}

// SetDraw replaces the draw callback (nil clears it) and requests a redraw.
func (cw *CanvasWidget) SetDraw(draw func(c render.Canvas, bounds geom.Rect)) {
	cw.draw = draw
	cw.Invalidate()
}

// SetOnEvent replaces the event callback (nil clears it).
func (cw *CanvasWidget) SetOnEvent(fn func(ev *Event) bool) { cw.onEvent = fn }

// Hovered reports whether the pointer is over the widget.
func (cw *CanvasWidget) Hovered() bool { return cw.hover }

// Focused reports whether the widget has keyboard focus.
func (cw *CanvasWidget) Focused() bool { return cw.focused }

// ToLocal converts an absolute position (such as Event.Pos) to one relative to
// the widget's top-left corner.
func (cw *CanvasWidget) ToLocal(p geom.Point) geom.Point {
	b := cw.Bounds()
	return geom.Point{X: p.X - b.X, Y: p.Y - b.Y}
}

// MinSize returns the size set with CanvasMinSize.
func (cw *CanvasWidget) MinSize() geom.Size { return cw.minSize }

// Focusable reports whether the widget takes keyboard focus (CanvasFocusable
// and enabled).
func (cw *CanvasWidget) Focusable() bool { return cw.focusable && cw.Enabled() }

// Draw calls the draw callback clipped to the widget's bounds, then the focus
// ring if the widget is focusable and focused.
func (cw *CanvasWidget) Draw(c render.Canvas) {
	b := cw.Bounds()
	if cw.draw != nil {
		c.PushClip(b)
		cw.draw(c, b)
		c.PopClip()
	}
	if cw.focusable && cw.focused {
		cw.strokeFocusable(c, b, cw.cornerRadius(), true)
	}
}

// HandleEvent tracks hover and focus, then offers the event to the callback.
func (cw *CanvasWidget) HandleEvent(ev *Event) bool {
	if !cw.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		cw.hover = true
	case EventPointerLeave:
		cw.hover = false
	case EventFocusGained:
		cw.focused = true
		cw.Invalidate()
	case EventFocusLost:
		cw.focused = false
		cw.Invalidate()
	}
	if cw.onEvent == nil {
		return false
	}
	return cw.onEvent(ev)
}
