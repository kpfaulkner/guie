package ui

import (
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// RadioGroup coordinates a set of RadioButtons so that exactly one is selected
// at a time. Buttons register themselves with the group when constructed.
type RadioGroup struct {
	members  []*RadioButton
	selected int // index of the selected member, or -1
	onChange func(index int)
}

// NewRadioGroup returns an empty group with nothing selected.
func NewRadioGroup() *RadioGroup { return &RadioGroup{selected: -1} }

// OnChange registers a handler called with the newly selected index.
func (g *RadioGroup) OnChange(fn func(index int)) { g.onChange = fn }

// Selected returns the selected index, or -1 if none.
func (g *RadioGroup) Selected() int { return g.selected }

func (g *RadioGroup) add(rb *RadioButton) int {
	g.members = append(g.members, rb)
	return len(g.members) - 1
}

func (g *RadioGroup) selectIndex(i int) {
	if i == g.selected {
		return
	}
	g.selected = i
	for j, m := range g.members {
		m.selected = j == i
	}
	if g.onChange != nil {
		g.onChange(i)
	}
}

// RadioButton is one option within a RadioGroup. Selecting it (via click or
// Space/Enter while focused) deselects the others in its group.
type RadioButton struct {
	BaseWidget
	label    string
	group    *RadioGroup
	index    int
	selected bool
	hover    bool
	focused  bool

	font render.FontFace
}

// NewRadioButton returns a RadioButton labeled label, registered with group.
func NewRadioButton(label string, group *RadioGroup) *RadioButton {
	rb := &RadioButton{BaseWidget: NewBase(), label: label, group: group}
	rb.index = group.add(rb)
	return rb
}

// IsSelected reports whether this button is the selected one in its group.
func (r *RadioButton) IsSelected() bool { return r.selected }

// Select makes this button the selected one in its group.
func (r *RadioButton) Select() { r.group.selectIndex(r.index) }

func (r *RadioButton) face() render.FontFace {
	if r.font != nil {
		return r.font
	}
	return r.appTheme().Font
}

// Focusable reports whether the button can take focus (only when enabled).
func (r *RadioButton) Focusable() bool { return r.Enabled() }

// MinSize returns the indicator plus gap plus label size.
func (r *RadioButton) MinSize() geom.Size {
	f := r.face()
	if f == nil {
		return geom.Size{}
	}
	side := f.Measure("Ag").H
	text := f.Measure(r.label)
	return geom.Size{W: side + indicatorGap + text.W, H: maxF(side, text.H)}
}

// Draw renders the circular indicator (filled when selected) and the label.
func (r *RadioButton) Draw(canvas render.Canvas) {
	f := r.face()
	if f == nil {
		return
	}
	b := r.Bounds()
	side := f.Measure("Ag").H
	radius := side / 2
	center := geom.Point{X: b.X + radius, Y: b.Y + b.H/2}

	if r.hover {
		canvas.FillCircle(center, radius, r.ColorOf(RoleSurface))
	}
	border := r.ColorOf(RoleBorder)
	if r.focused {
		border = r.ColorOf(RoleAccent)
	}
	canvas.StrokeCircle(center, radius, border, 1)
	if r.selected {
		canvas.FillCircle(center, radius*0.5, r.ColorOf(RoleAccent))
	}

	canvas.DrawText(r.label, geom.Point{X: b.X + side + indicatorGap, Y: vCenterY(f, b.Y, b.H)}, f, r.ColorOf(RoleText))
}

// HandleEvent selects the button on click or Space/Enter and tracks hover/focus.
func (r *RadioButton) HandleEvent(ev *Event) bool {
	if !r.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		r.hover = true
		return true
	case EventPointerLeave:
		r.hover = false
		return true
	case EventClick:
		r.Select()
		return true
	case EventFocusGained:
		r.focused = true
		return true
	case EventFocusLost:
		r.focused = false
		return true
	case EventKeyDown:
		if ev.Key == render.KeySpace || ev.Key == render.KeyEnter {
			r.Select()
			return true
		}
	}
	return false
}
