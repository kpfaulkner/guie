package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

const (
	dropdownArrowW   = 18
	dropdownMaxRows  = 8
	dropdownVPadding = 5
)

// DropdownCombo is a collapsed selector showing the current choice. Clicking it
// opens a popup List of options below it; choosing one updates the selection,
// fires OnChange and closes the popup. Clicking outside or pressing Escape also
// closes it.
type DropdownCombo struct {
	BaseWidget
	options     []string
	selected    int // -1 = none
	placeholder string
	hover       bool
	focused     bool
	open        bool
	popup       *Popup
	onChange    func(int)
	font        render.FontFace
}

// DropdownOption configures a DropdownCombo.
type DropdownOption func(*DropdownCombo)

// DropdownSelected sets the initially selected index.
func DropdownSelected(i int) DropdownOption { return func(d *DropdownCombo) { d.selected = i } }

// DropdownPlaceholder sets the text shown when nothing is selected.
func DropdownPlaceholder(s string) DropdownOption {
	return func(d *DropdownCombo) { d.placeholder = s }
}

// NewDropdown returns a DropdownCombo over options, configured by opts.
func NewDropdown(options []string, opts ...DropdownOption) *DropdownCombo {
	d := &DropdownCombo{BaseWidget: NewBase(), options: options, selected: -1, placeholder: "Select..."}
	for _, o := range opts {
		o(d)
	}
	return d
}

// OnSelect registers the handler invoked with the chosen index.
func (d *DropdownCombo) OnSelect(fn func(int)) { d.onChange = fn }

// Selected returns the selected index, or -1.
func (d *DropdownCombo) Selected() int { return d.selected }

func (d *DropdownCombo) face() render.FontFace {
	if d.font != nil {
		return d.font
	}
	return d.appTheme().Font
}

// SetFont overrides the dropdown's font face (nil falls back to the theme font).
func (d *DropdownCombo) SetFont(f render.FontFace) {
	d.font = f
	d.Invalidate()
}

// Focusable reports whether the dropdown can take focus (only when enabled).
func (d *DropdownCombo) Focusable() bool { return d.Enabled() }

// MinSize fits the widest option (or placeholder) plus the arrow and padding.
func (d *DropdownCombo) MinSize() geom.Size {
	f := d.face()
	if f == nil {
		return geom.Size{}
	}
	w := f.Measure(d.placeholder).W
	for _, o := range d.options {
		w = maxF(w, f.Measure(o).W)
	}
	h := f.Measure("Ag").H
	return geom.Size{W: w + dropdownArrowW + 2*listRowPad, H: h + 2*dropdownVPadding}
}

func (d *DropdownCombo) label() string {
	if d.selected >= 0 && d.selected < len(d.options) {
		return d.options[d.selected]
	}
	return d.placeholder
}

// Draw renders the box, current label and a chevron, accented when open/focused.
func (d *DropdownCombo) Draw(canvas render.Canvas) {
	f := d.face()
	if f == nil {
		return
	}
	b := d.Bounds()
	rad := d.cornerRadius()
	canvas.FillRoundRect(b, rad, d.ColorOf(RoleSurface))
	border := d.ColorOf(RoleBorder)
	if d.focused || d.open {
		border = d.ColorOf(RoleAccent)
	}
	canvas.StrokeRoundRect(b, rad, border, 1)

	textColor := d.ColorOf(RoleText)
	if d.selected < 0 {
		textColor = d.ColorOf(RoleTextMuted)
	}
	canvas.DrawText(d.label(), geom.Point{X: b.X + listRowPad, Y: vCenterY(f, b.Y, b.H)}, f, textColor)

	// Chevron in the arrow area on the right.
	cx := b.X + b.W - dropdownArrowW/2 - listRowPad
	cy := b.Y + b.H/2
	const aw = 4
	canvas.DrawLine(geom.Point{X: cx - aw, Y: cy - aw/2}, geom.Point{X: cx, Y: cy + aw/2}, d.ColorOf(RoleText), 2)
	canvas.DrawLine(geom.Point{X: cx, Y: cy + aw/2}, geom.Point{X: cx + aw, Y: cy - aw/2}, d.ColorOf(RoleText), 2)
}

// HandleEvent toggles the popup on click/Enter/Space and tracks hover/focus.
func (d *DropdownCombo) HandleEvent(ev *Event) bool {
	if !d.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		d.hover = true
		return true
	case EventPointerLeave:
		d.hover = false
		return true
	case EventClick:
		d.toggle()
		return true
	case EventFocusGained:
		d.focused = true
		return true
	case EventFocusLost:
		d.focused = false
		return true
	case EventKeyDown:
		if ev.Key == render.KeySpace || ev.Key == render.KeyEnter {
			d.toggle()
			return true
		}
	}
	return false
}

func (d *DropdownCombo) toggle() {
	if d.open {
		d.ctx.close(d.popup)
		return
	}
	d.openList()
}

// openList builds a popup List positioned directly below the dropdown.
func (d *DropdownCombo) openList() {
	if len(d.options) == 0 || d.ctx == nil {
		return
	}
	list := NewList(d.options, ListSelected(d.selected))
	list.font = d.face() // resolved font, so RowHeight works before the list mounts
	list.onSelect = func(i int) {
		d.selected = i
		if d.onChange != nil {
			d.onChange(i)
		}
		d.ctx.close(d.popup)
	}

	b := d.Bounds()
	rows := len(d.options)
	if rows > dropdownMaxRows {
		rows = dropdownMaxRows
	}
	height := list.RowHeight() * float64(rows)
	bounds := geom.Rect{X: b.X, Y: b.Y + b.H, W: b.W, H: height}

	d.popup = NewPopup(list, bounds, func() { d.open = false })
	d.ctx.open(d.popup)
	d.open = true
}
