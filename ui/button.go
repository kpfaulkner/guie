package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Button is a clickable widget. It tracks hover and pressed state for visual
// feedback and invokes OnClick when a left-click is released over it.
type Button struct {
	BaseWidget

	Text    string
	OnClick func()

	// Colors for the three visual states. Zero values fall back to the
	// package defaults at draw time.
	Background        color.Color
	HoverBackground   color.Color
	PressedBackground color.Color

	hovered bool
	pressed bool
}

// NewButton returns a button occupying r with the given label.
func NewButton(r Rect, text string) *Button {
	return &Button{
		BaseWidget:        NewBase(r),
		Text:              text,
		Background:        DefaultButton,
		HoverBackground:   DefaultButtonHover,
		PressedBackground: DefaultButtonPressed,
	}
}

// Draw renders the button background in its current state with a centered-ish
// label.
func (b *Button) Draw(dst *ebiten.Image, origin Point) {
	abs := b.bounds.Add(origin)

	bg := b.Background
	switch {
	case b.pressed:
		bg = b.PressedBackground
	case b.hovered:
		bg = b.HoverBackground
	}
	if bg == nil {
		bg = DefaultButton
	}

	vector.FillRect(dst, float32(abs.X), float32(abs.Y), float32(abs.W), float32(abs.H), bg, false)

	// The debug font is 6px wide / 16px tall per character; approximate a
	// centered label.
	tx := abs.X + (abs.W-len(b.Text)*6)/2
	ty := abs.Y + (abs.H-16)/2
	ebitenutil.DebugPrintAt(dst, b.Text, tx, ty)
}

// HandleEvent updates hover/pressed state and fires OnClick on release.
func (b *Button) HandleEvent(ev *Event, origin Point) bool {
	abs := b.bounds.Add(origin)
	inside := abs.Contains(ev.Pos)

	switch ev.Type {
	case MouseMove:
		b.hovered = inside
	case MouseDown:
		if inside && ev.Button == ebiten.MouseButtonLeft {
			b.pressed = true
			return true
		}
	case MouseUp:
		if b.pressed && ev.Button == ebiten.MouseButtonLeft {
			b.pressed = false
			if inside && b.OnClick != nil {
				b.OnClick()
			}
			return inside
		}
	}
	return false
}
