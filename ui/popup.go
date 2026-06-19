package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// Popup is a transient overlay (dropdown list, menu) drawn above the main
// widget tree. Its content is positioned at Bounds in surface coordinates.
// Popups are managed as a stack by the App: opening pushes, and clicking
// outside all popups (or pressing Escape) dismisses them.
type Popup struct {
	content Widget
	bounds  geom.Rect
	onClose func()
}

// NewPopup returns a Popup that shows content at the given absolute bounds.
// onClose, if non-nil, is called when the popup is dismissed.
func NewPopup(content Widget, bounds geom.Rect, onClose func()) *Popup {
	return &Popup{content: content, bounds: bounds, onClose: onClose}
}

// openPopup mounts the popup's content, lays it out at its bounds and pushes it
// onto the overlay stack.
func (a *App) openPopup(p *Popup) {
	if p == nil || p.content == nil {
		return
	}
	p.content.mount(nil, a.ctx)
	p.content.SetBounds(p.bounds)
	p.content.Layout()
	a.overlays = append(a.overlays, p)
}

// closePopup removes p (and any popups stacked above it), invoking each
// onClose. Unknown popups are ignored.
func (a *App) closePopup(p *Popup) {
	idx := -1
	for i, ov := range a.overlays {
		if ov == p {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	a.closeFrom(idx)
}

// closeTop dismisses the top-most popup, if any.
func (a *App) closeTop() {
	if len(a.overlays) > 0 {
		a.closeFrom(len(a.overlays) - 1)
	}
}

// closeAll dismisses every open popup.
func (a *App) closeAll() {
	if len(a.overlays) > 0 {
		a.closeFrom(0)
	}
}

// closeFrom removes overlays[idx:] in top-down order, calling each onClose.
func (a *App) closeFrom(idx int) {
	for i := len(a.overlays) - 1; i >= idx; i-- {
		ov := a.overlays[i]
		if ov.onClose != nil {
			ov.onClose()
		}
	}
	a.overlays = a.overlays[:idx]
	if a.hovered != nil {
		a.hovered = nil
	}
	a.pressTarget = nil
}

// layoutOverlays re-applies each popup's bounds and lays out its content. Called
// during the layout pass so popups reflow when their content changes.
func (a *App) layoutOverlays() {
	for _, ov := range a.overlays {
		ov.content.SetBounds(ov.bounds)
		ov.content.Layout()
	}
}

// drawOverlays paints every popup on top of the main tree, in stack order.
func (a *App) drawOverlays(c render.Canvas) {
	for _, ov := range a.overlays {
		if ov.content.Visible() {
			ov.content.Draw(c)
		}
	}
}

// overlayHit returns the top-most widget under pos within any popup, and
// whether pos fell inside a popup at all.
func (a *App) overlayHit(pos geom.Point) (Widget, bool) {
	for i := len(a.overlays) - 1; i >= 0; i-- {
		if h := hitTest(a.overlays[i].content, pos); h != nil {
			return h, true
		}
	}
	return nil, false
}
