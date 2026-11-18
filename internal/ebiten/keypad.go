package ebitenbackend

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/guie/render"
)

// keypadNav is what the numeric keypad's keys mean with NumLock off: the keys
// printed under the digits on a PC keyboard. Keypad 5 has no such key and is
// left out. With NumLock on none of them is a key: the digit arrives as text
// (InputState.Runes), as on any other key that types.
//
// EBiten reports the keypad by position, KeyNumpad7 whatever NumLock says, so
// without this map NumLock off left the keypad doing nothing at all.
var keypadNav = map[ebiten.Key]render.Key{
	ebiten.KeyNumpad7:       render.KeyHome,
	ebiten.KeyNumpad8:       render.KeyUp,
	ebiten.KeyNumpad9:       render.KeyPageUp,
	ebiten.KeyNumpad4:       render.KeyLeft,
	ebiten.KeyNumpad6:       render.KeyRight,
	ebiten.KeyNumpad1:       render.KeyEnd,
	ebiten.KeyNumpad2:       render.KeyDown,
	ebiten.KeyNumpad3:       render.KeyPageDown,
	ebiten.KeyNumpad0:       render.KeyInsert,
	ebiten.KeyNumpadDecimal: render.KeyDelete,
}

// keypad maps the keypad keys by NumLock, and keeps for each key held the
// meaning it had when it went down. A key that changed meaning while held (the
// user toggled NumLock) would otherwise be released under another name, and a
// widget waiting for its release, such as a key repeat, would never see it.
//
// The zero value is ready to use. It is only touched by the game loop.
type keypad struct {
	held map[ebiten.Key]render.Key // KeyUnknown: held, but no key (NumLock was on)
}

// keys maps one frame's key lists. numLock is the state this frame. Keys that
// are not on the keypad map through keyMap as before.
func (k *keypad) keys(down, pressed, released []ebiten.Key, numLock bool) (d, p, r []render.Key) {
	if k.held == nil {
		k.held = map[ebiten.Key]render.Key{}
	}
	meaning := func(e ebiten.Key) render.Key {
		if m, ok := k.held[e]; ok {
			return m
		}
		m := render.KeyUnknown
		if !numLock {
			m = keypadNav[e]
		}
		k.held[e] = m
		return m
	}
	mapList := func(keys []ebiten.Key, pressedNow bool) []render.Key {
		var out []render.Key
		for _, e := range keys {
			if _, onPad := keypadNav[e]; onPad {
				if pressedNow {
					delete(k.held, e) // a new press takes the state now
				}
				if m := meaning(e); m != render.KeyUnknown {
					out = append(out, m)
				}
				continue
			}
			if rk, ok := keyMap[e]; ok {
				out = append(out, rk)
			}
		}
		return out
	}
	// Pressed first, so a key that went down this frame takes this frame's
	// state, then down, then released, which forgets the key.
	p = mapList(pressed, true)
	d = mapList(down, false)
	r = mapList(released, false)
	for _, e := range released {
		delete(k.held, e)
	}
	// A key no longer down and not reported released (focus lost) is forgotten
	// too, so the map cannot grow or go stale.
	for e := range k.held {
		if !contains(down, e) {
			delete(k.held, e)
		}
	}
	return d, p, r
}

func contains(keys []ebiten.Key, e ebiten.Key) bool {
	for _, k := range keys {
		if k == e {
			return true
		}
	}
	return false
}
