package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Container is a widget that holds and lays out child widgets. Children are
// positioned with absolute bounds relative to the container's content area
// (its own origin). Children are drawn in insertion order, so later children
// appear on top; events are dispatched in reverse order, so the topmost child
// gets first chance to consume an event.
type Container struct {
	BaseWidget

	// Background, if non-nil, fills the container's rectangle before children
	// are drawn. A nil Background leaves the container transparent.
	Background color.Color

	children []Widget
}

// NewContainer returns an empty container occupying r.
func NewContainer(r Rect) *Container {
	return &Container{BaseWidget: NewBase(r)}
}

// Add appends a child widget to the container.
func (c *Container) Add(w Widget) {
	c.children = append(c.children, w)
}

// Children returns the container's child widgets.
func (c *Container) Children() []Widget { return c.children }

// Update advances every child.
func (c *Container) Update() error {
	for _, w := range c.children {
		if err := w.Update(); err != nil {
			return err
		}
	}
	return nil
}

// Draw fills the optional background and renders every visible child.
func (c *Container) Draw(dst *ebiten.Image, origin Point) {
	abs := c.bounds.Add(origin)
	if c.Background != nil {
		vector.FillRect(dst, float32(abs.X), float32(abs.Y), float32(abs.W), float32(abs.H), c.Background, false)
	}
	childOrigin := abs.Origin()
	for _, w := range c.children {
		if w.IsVisible() {
			w.Draw(dst, childOrigin)
		}
	}
}

// HandleEvent dispatches an event to the children, topmost first.
func (c *Container) HandleEvent(ev *Event, origin Point) bool {
	childOrigin := c.bounds.Add(origin).Origin()
	for i := len(c.children) - 1; i >= 0; i-- {
		w := c.children[i]
		if !w.IsVisible() {
			continue
		}
		if w.HandleEvent(ev, childOrigin) {
			return true
		}
	}
	return false
}
