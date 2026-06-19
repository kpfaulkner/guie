package ui

import (
	"image/color"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// scrimColor dims the background behind a modal popup.
var scrimColor = color.RGBA{R: 0, G: 0, B: 0, A: 0xA0}

// Popup is a transient overlay (dropdown list, menu, dialog) drawn above the
// main widget tree. Its content is positioned at Bounds in surface coordinates.
// Popups are managed as a stack by the App.
//
// A non-modal popup (dropdown, menu) is dismissed by clicking outside it or
// pressing Escape. A modal popup (dialog) draws a full-screen scrim, blocks all
// input to the widgets behind it, and is not dismissed by outside clicks — only
// programmatically (e.g. by a dialog button) or by Escape.
type Popup struct {
	content Widget
	bounds  geom.Rect
	onClose func()
	modal   bool
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
	p.content.mount(p.content, nil, a.ctx)
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

// drawOverlays paints every popup on top of the main tree, in stack order. A
// modal popup is preceded by a full-screen scrim.
func (a *App) drawOverlays(c render.Canvas) {
	for _, ov := range a.overlays {
		if ov.modal {
			c.FillRect(geom.Rect{W: a.surfaceSize.W, H: a.surfaceSize.H}, scrimColor)
		}
		if a.shadows {
			drawShadow(c, ov.bounds, a.theme.CornerRadius)
		}
		if ov.content.Visible() {
			ov.content.Draw(c)
		}
	}
}

// modalActive returns the top-most popup if it is modal, else nil.
func (a *App) modalActive() *Popup {
	if n := len(a.overlays); n > 0 && a.overlays[n-1].modal {
		return a.overlays[n-1]
	}
	return nil
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

const minModalWidth = 240

// ShowModal centers content on screen and shows it as a modal popup: a scrim
// dims the background and all input behind the dialog is blocked. It returns the
// Popup handle; close it with Close (or from within content via the framework).
func (a *App) ShowModal(content Widget) *Popup {
	p := &Popup{content: content, modal: true}
	content.mount(content, nil, a.ctx) // mount first so MinSize can measure text
	size := content.MinSize()
	if size.W < minModalWidth {
		size.W = minModalWidth
	}
	x := (a.surfaceSize.W - size.W) / 2
	y := (a.surfaceSize.H - size.H) / 2
	p.bounds = geom.Rect{X: x, Y: y, W: size.W, H: size.H}
	content.SetBounds(p.bounds)
	content.Layout()
	a.overlays = append(a.overlays, p)
	return p
}

// Close dismisses the popup (calling its onClose). Safe to call from a dialog
// button handler.
func (a *App) Close(p *Popup) { a.closePopup(p) }

// DialogButton describes one button in a message dialog: a label and an optional
// handler run (before the dialog closes) when it is chosen.
type DialogButton struct {
	Label   string
	OnClick func()
}

// ShowMessage builds and shows a modal dialog with a title, a message and a row
// of buttons. Each button runs its handler (if any) and closes the dialog. If
// no buttons are given, a single "OK" button is added.
func (a *App) ShowMessage(title, message string, buttons ...DialogButton) *Popup {
	if len(buttons) == 0 {
		buttons = []DialogButton{{Label: "OK"}}
	}
	pal := a.theme.Palette

	panel := NewContainer()
	panel.SetBackground(pal.Surface)
	panel.SetBorder(pal.Border, 1)
	panel.SetCornerRadius(a.theme.CornerRadius)
	panel.SetLayout(VBox(12))
	panel.SetPadding(geom.UniformInsets(16))

	panel.Add(NewLabel(title))
	panel.Add(NewLabel(message, LabelColor(pal.TextMuted)))

	var popup *Popup
	row := NewContainer()
	row.SetLayout(HBox(10))
	for _, b := range buttons {
		bb := b
		btn := NewButton(bb.Label)
		btn.OnClick(func() {
			if bb.OnClick != nil {
				bb.OnClick()
			}
			a.closePopup(popup)
		})
		row.Add(btn)
	}
	panel.Add(row, Align(geom.AlignEnd), Weight(1))

	popup = a.ShowModal(panel)
	return popup
}
