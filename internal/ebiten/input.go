package ebitenbackend

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/render"
)

// primaryIsMeta reports whether the platform's primary shortcut modifier is the
// Meta/Command key (macOS) rather than Control (everywhere else).
var primaryIsMeta = runtime.GOOS == "darwin"

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
//
// scale is the current device scale factor. EBiten reports the cursor in the
// coordinate space of the game screen, which the driver sizes in physical
// pixels so HiDPI rendering stays crisp, so the position must be divided back
// down to the logical pixels the framework lays out and hit-tests in. On a
// 1x display this is a no-op; at 2x an undivided position lands at double the
// intended coordinates and every hit test misses.
func pollInput(scale float64) render.InputState {
	if scale <= 0 {
		scale = 1
	}
	cx, cy := ebiten.CursorPosition()
	in := render.InputState{
		MousePos: geom.Point{X: float64(cx) / scale, Y: float64(cy) / scale},
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
	// EBiten reports dropped files for the single tick after the OS drop, then
	// clears them, which is the edge InputState.DroppedFiles describes. A nil FS
	// (no drop) must stay nil rather than becoming an empty non-nil FS, so the
	// framework can test it directly.
	if dropped := ebiten.DroppedFiles(); dropped != nil {
		in.DroppedFiles = dropped
	}

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
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	meta := ebiten.IsKeyPressed(ebiten.KeyMeta)
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		m |= render.ModifierSet(render.ModShift)
	}
	if ctrl {
		m |= render.ModifierSet(render.ModControl)
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) {
		m |= render.ModifierSet(render.ModAlt)
	}
	if meta {
		m |= render.ModifierSet(render.ModMeta)
	}
	if (primaryIsMeta && meta) || (!primaryIsMeta && ctrl) {
		m |= render.ModifierSet(render.ModPrimary)
	}
	return m
}
