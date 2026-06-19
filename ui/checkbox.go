package ui

import (
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

// indicatorGap is the space between a checkbox/radio indicator and its label.
const indicatorGap = 8

// Checkbox is a labeled boolean toggle. It is focusable and toggles on click or
// when Space/Enter is pressed while focused, invoking OnChange with the new
// value.
type Checkbox struct {
	BaseWidget
	label    string
	checked  bool
	hover    bool
	focused  bool
	onChange func(bool)

	font render.FontFace // nil → theme font
}

// CheckboxOption configures a Checkbox.
type CheckboxOption func(*Checkbox)

// Checked sets the initial checked state.
func Checked(v bool) CheckboxOption { return func(c *Checkbox) { c.checked = v } }

// NewCheckbox returns a Checkbox with the given label.
func NewCheckbox(label string, opts ...CheckboxOption) *Checkbox {
	c := &Checkbox{BaseWidget: NewBase(), label: label}
	for _, o := range opts {
		o(c)
	}
	return c
}

// OnChange registers the handler invoked when the checked state changes.
func (c *Checkbox) OnChange(fn func(bool)) { c.onChange = fn }

// Checked reports whether the box is checked.
func (c *Checkbox) IsChecked() bool { return c.checked }

// SetChecked sets the checked state and fires OnChange if it changed.
func (c *Checkbox) SetChecked(v bool) {
	if c.checked == v {
		return
	}
	c.checked = v
	if c.onChange != nil {
		c.onChange(v)
	}
}

func (c *Checkbox) face() render.FontFace {
	if c.font != nil {
		return c.font
	}
	return c.appTheme().Font
}

// Focusable reports whether the checkbox can take focus (only when enabled).
func (c *Checkbox) Focusable() bool { return c.Enabled() }

// MinSize returns the indicator plus gap plus label size.
func (c *Checkbox) MinSize() geom.Size {
	f := c.face()
	if f == nil {
		return geom.Size{}
	}
	side := f.Measure("Ag").H
	text := f.Measure(c.label)
	return geom.Size{W: side + indicatorGap + text.W, H: maxF(side, text.H)}
}

func (c *Checkbox) toggle() { c.SetChecked(!c.checked) }

// Draw renders the box, the check mark when checked, and the label.
func (c *Checkbox) Draw(canvas render.Canvas) {
	f := c.face()
	if f == nil {
		return
	}
	b := c.Bounds()
	side := f.Measure("Ag").H
	box := geom.Rect{X: b.X, Y: b.Y + (b.H-side)/2, W: side, H: side}

	if c.checked {
		canvas.FillRect(box, c.ColorOf(RolePrimary))
	} else if c.hover {
		canvas.FillRect(box, c.ColorOf(RoleSurface))
	}
	border := c.ColorOf(RoleBorder)
	if c.focused {
		border = c.ColorOf(RoleAccent)
	}
	canvas.StrokeRect(box, border, 1)

	if c.checked {
		col := c.ColorOf(RoleOnPrimary)
		canvas.DrawLine(
			geom.Point{X: box.X + box.W*0.22, Y: box.Y + box.H*0.52},
			geom.Point{X: box.X + box.W*0.42, Y: box.Y + box.H*0.72},
			col, 2,
		)
		canvas.DrawLine(
			geom.Point{X: box.X + box.W*0.42, Y: box.Y + box.H*0.72},
			geom.Point{X: box.X + box.W*0.80, Y: box.Y + box.H*0.28},
			col, 2,
		)
	}

	ts := f.Measure(c.label)
	canvas.DrawText(c.label, geom.Point{X: box.X + side + indicatorGap, Y: b.Y + (b.H-ts.H)/2}, f, c.ColorOf(RoleText))
}

// HandleEvent toggles on click or Space/Enter and tracks hover/focus.
func (c *Checkbox) HandleEvent(ev *Event) bool {
	if !c.Enabled() {
		return false
	}
	switch ev.Type {
	case EventPointerEnter:
		c.hover = true
		return true
	case EventPointerLeave:
		c.hover = false
		return true
	case EventClick:
		c.toggle()
		return true
	case EventFocusGained:
		c.focused = true
		return true
	case EventFocusLost:
		c.focused = false
		return true
	case EventKeyDown:
		if ev.Key == render.KeySpace || ev.Key == render.KeyEnter {
			c.toggle()
			return true
		}
	}
	return false
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
