package ui

import (
	"image/color"
	"math"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// Container is a widget that holds and draws child widgets. It is the basic
// grouping element and the root of a window's content.
//
// In step 2 it draws its children at their assigned bounds and clips them to
// its content area; the layout engine (step 3) will add automatic positioning
// by attaching a layout manager.
type Container struct {
	BaseWidget
	children    []Widget
	data        []LayoutData // per-child layout params, parallel to children
	layout      Layout       // optional; nil means children keep their bounds
	background  color.Color  // optional fill; nil means transparent
	padding     geom.Insets
	borderColor color.Color // optional outline; nil means none
	borderWidth float64
}

// NewContainer returns an empty, visible Container.
func NewContainer() *Container {
	return &Container{BaseWidget: NewBase()}
}

// SetLayout sets the layout manager that positions the children.
func (c *Container) SetLayout(l Layout) {
	c.layout = l
	c.Invalidate()
}

// SetBackground sets the fill color drawn behind the children. Nil is transparent.
func (c *Container) SetBackground(col color.Color) {
	c.background = col
	c.Invalidate()
}

// SetPadding reserves inner space around the children.
func (c *Container) SetPadding(in geom.Insets) {
	c.padding = in
	c.Invalidate()
}

// SetBorder draws an outline of the given color and width around the container.
// A nil color removes the border.
func (c *Container) SetBorder(col color.Color, width float64) {
	c.borderColor = col
	c.borderWidth = width
}

// Background returns the container's fill color (nil means transparent).
func (c *Container) Background() color.Color { return c.background }

// BorderColor returns the container's border color (nil means no border), and
// its width.
func (c *Container) BorderColor() (color.Color, float64) { return c.borderColor, c.borderWidth }

// Add appends a child widget with optional per-child layout parameters,
// mounting it immediately if the container is already part of a mounted tree.
func (c *Container) Add(w Widget, opts ...ItemOption) {
	d := defaultLayoutData()
	for _, o := range opts {
		o(&d)
	}
	c.children = append(c.children, w)
	c.data = append(c.data, d)
	if c.ctx != nil {
		w.mount(w, c.self, c.ctx)
	}
	c.Invalidate()
}

// Children returns the container's child widgets.
func (c *Container) Children() []Widget { return c.children }

// items pairs each child with its layout data for the layout manager.
func (c *Container) items() []Item {
	items := make([]Item, len(c.children))
	for i, ch := range c.children {
		items[i] = Item{Widget: ch, Data: c.data[i]}
	}
	return items
}

// ContentRect returns the area available to children: Bounds inset by padding.
func (c *Container) ContentRect() geom.Rect {
	return c.Bounds().Inset(c.padding)
}

// MinSize returns the size needed to enclose the children plus padding. With a
// layout manager it defers to the layout's measurement; otherwise it takes the
// largest child extents on each axis.
func (c *Container) MinSize() geom.Size {
	var content geom.Size
	if c.layout != nil {
		content = c.layout.Measure(c.items())
	} else {
		for _, ch := range c.children {
			m := ch.MinSize()
			content.W = math.Max(content.W, m.W)
			content.H = math.Max(content.H, m.H)
		}
	}
	return geom.Size{
		W: content.W + c.padding.Left + c.padding.Right,
		H: content.H + c.padding.Top + c.padding.Bottom,
	}
}

// Layout arranges the children via the layout manager (if any), then recurses
// so nested containers position their own children. Without a layout manager
// children keep their assigned bounds.
func (c *Container) Layout() {
	if c.layout != nil {
		c.layout.Arrange(c.items(), c.ContentRect())
	}
	for _, ch := range c.children {
		ch.Layout()
	}
}

// Draw paints the background, then each visible child clipped to the content
// area.
func (c *Container) Draw(canvas render.Canvas) {
	if c.background != nil {
		canvas.FillRect(c.Bounds(), c.background)
	}
	canvas.PushClip(c.ContentRect())
	for _, ch := range c.children {
		if ch.Visible() {
			ch.Draw(canvas)
		}
	}
	canvas.PopClip()

	if c.borderColor != nil && c.borderWidth > 0 {
		canvas.StrokeRect(c.Bounds(), c.borderColor, c.borderWidth)
	}
}

// mount attaches the container and all of its current children to the tree.
func (c *Container) mount(self, parent Widget, ctx *treeContext) {
	c.BaseWidget.mount(self, parent, ctx)
	for _, ch := range c.children {
		ch.mount(ch, self, ctx)
	}
}
