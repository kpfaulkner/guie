package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// accelerator binds a key chord to an action.
type accelerator struct {
	key    render.Key
	mods   render.ModifierSet // normalized (Primary/Shift/Alt only)
	action func()
}

// AddAccelerator registers a global keyboard shortcut: when key is pressed with
// exactly mods held, action runs instead of the key reaching the focused widget.
//
// Use render.ModPrimary for the platform shortcut key (Command on macOS, Control
// elsewhere), optionally combined with render.ModShift / render.ModAlt, e.g.
//
//	app.AddAccelerator(render.KeyS, render.ModifierSet(render.ModPrimary), save)
//	app.AddAccelerator(render.KeyZ, render.ModifierSet(render.ModPrimary|render.ModShift), redo)
//
// Matching is exact over the Primary/Shift/Alt modifiers, so Primary+S does not
// fire for Primary+Shift+S. An accelerator takes precedence over the focused
// widget, so avoid binding chords a focused widget needs (e.g. Primary+C, which
// text widgets use for copy) unless that is the intent.
func (a *App) AddAccelerator(key render.Key, mods render.ModifierSet, action func()) {
	if action == nil {
		return
	}
	a.accels = append(a.accels, accelerator{key: key, mods: normalizeMods(mods), action: action})
}

// normalizeMods reduces a modifier set to the bits accelerators compare on:
// Primary (Cmd/Ctrl), Shift and Alt. The redundant concrete Control/Meta bit is
// dropped so a ModPrimary accelerator matches on every platform.
func normalizeMods(m render.ModifierSet) render.ModifierSet {
	var n render.ModifierSet
	if m.Has(render.ModShift) {
		n |= render.ModifierSet(render.ModShift)
	}
	if m.Has(render.ModAlt) {
		n |= render.ModifierSet(render.ModAlt)
	}
	if m.Has(render.ModPrimary) {
		n |= render.ModifierSet(render.ModPrimary)
	}
	return n
}

// runAccelerator runs the first accelerator matching key+mods and reports
// whether one fired (so the key is not also delivered to the focused widget).
func (a *App) runAccelerator(key render.Key, mods render.ModifierSet) bool {
	nm := normalizeMods(mods)
	for _, ac := range a.accels {
		if ac.key == key && ac.mods == nm {
			ac.action()
			return true
		}
	}
	return false
}

// contextTarget walks up from w to the first widget with a context menu, or nil.
func contextTarget(w Widget) Widget {
	for ; w != nil; w = w.Parent() {
		if len(w.ContextMenu()) > 0 {
			return w
		}
	}
	return nil
}

// ShowContextMenu opens a popup menu of items at the given surface position,
// clamped to stay on screen. It returns the popup (nil if there are no items).
// Choosing an item runs its Action and closes the menu; clicking elsewhere or
// pressing Escape dismisses it.
func (a *App) ShowContextMenu(at geom.Point, items ...MenuItem) *Popup {
	if len(items) == 0 {
		return nil
	}
	var popup *Popup
	panel, sz := menuPanel(a.theme, nil, items, func(action func()) {
		a.closePopup(popup)
		if action != nil {
			action()
		}
	})

	x, y := at.X, at.Y
	if x+sz.W > a.surfaceSize.W {
		x = a.surfaceSize.W - sz.W
	}
	if y+sz.H > a.surfaceSize.H {
		y = a.surfaceSize.H - sz.H
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	popup = NewPopup(panel, geom.Rect{X: x, Y: y, W: sz.W, H: sz.H}, nil)
	a.openPopup(popup)
	return popup
}
