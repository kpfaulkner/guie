package ebitenbackend

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/render"
)

var mouseButtons = []struct {
	eb ebiten.MouseButton
	rb render.MouseButton
}{
	{ebiten.MouseButtonLeft, render.MouseLeft},
	{ebiten.MouseButtonMiddle, render.MouseMiddle},
	{ebiten.MouseButtonRight, render.MouseRight},
}

// pollInput captures the current frame's EBiten input as a backend-neutral
// render.InputState.
func pollInput() render.InputState {
	cx, cy := ebiten.CursorPosition()
	in := render.InputState{
		MousePos: geom.Point{X: float64(cx), Y: float64(cy)},
	}

	for _, m := range mouseButtons {
		if ebiten.IsMouseButtonPressed(m.eb) {
			in.MouseDown = in.MouseDown.Set(m.rb)
		}
		if inpututil.IsMouseButtonJustPressed(m.eb) {
			in.MousePressed = in.MousePressed.Set(m.rb)
		}
		if inpututil.IsMouseButtonJustReleased(m.eb) {
			in.MouseReleased = in.MouseReleased.Set(m.rb)
		}
	}

	wx, wy := ebiten.Wheel()
	in.WheelDelta = geom.Point{X: wx, Y: wy}

	in.KeysDown = mapKeys(inpututil.AppendPressedKeys(nil))
	in.KeysPressed = mapKeys(inpututil.AppendJustPressedKeys(nil))
	in.KeysReleased = mapKeys(inpututil.AppendJustReleasedKeys(nil))
	in.Runes = ebiten.AppendInputChars(nil)
	in.Modifiers = pollModifiers()

	return in
}

// mapKeys translates a slice of EBiten keys to render.Key values, dropping any
// that have no mapping.
func mapKeys(keys []ebiten.Key) []render.Key {
	var out []render.Key
	for _, k := range keys {
		if rk, ok := keyMap[k]; ok {
			out = append(out, rk)
		}
	}
	return out
}

func pollModifiers() render.ModifierSet {
	var m render.ModifierSet
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		m |= render.ModifierSet(render.ModShift)
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		m |= render.ModifierSet(render.ModControl)
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		m |= render.ModifierSet(render.ModAlt)
	}
	if ebiten.IsKeyPressed(ebiten.KeyMeta) {
		m |= render.ModifierSet(render.ModMeta)
	}
	return m
}
