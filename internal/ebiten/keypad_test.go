package ebitenbackend

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/guie/render"
)

func TestKeypadNumLockOff(t *testing.T) {
	for e, want := range map[ebiten.Key]render.Key{
		ebiten.KeyNumpad7: render.KeyHome, ebiten.KeyNumpad8: render.KeyUp, ebiten.KeyNumpad9: render.KeyPageUp,
		ebiten.KeyNumpad4: render.KeyLeft, ebiten.KeyNumpad6: render.KeyRight,
		ebiten.KeyNumpad1: render.KeyEnd, ebiten.KeyNumpad2: render.KeyDown, ebiten.KeyNumpad3: render.KeyPageDown,
		ebiten.KeyNumpad0: render.KeyInsert, ebiten.KeyNumpadDecimal: render.KeyDelete,
	} {
		var kp keypad
		keys := []ebiten.Key{e}
		d, p, r := kp.keys(keys, keys, nil, false)
		if !reflect.DeepEqual(d, []render.Key{want}) || !reflect.DeepEqual(p, []render.Key{want}) || r != nil {
			t.Errorf("%v with NumLock off: down %v pressed %v released %v, want %v", e, d, p, r, want)
		}
		_, _, r = kp.keys(nil, nil, keys, false)
		if !reflect.DeepEqual(r, []render.Key{want}) {
			t.Errorf("%v released as %v, want %v", e, r, want)
		}
	}
}

func TestKeypadNumLockOn(t *testing.T) {
	// The digit comes as text; no key is reported, so it is not typed twice.
	var kp keypad
	keys := []ebiten.Key{ebiten.KeyNumpad7, ebiten.KeyNumpad5, ebiten.KeyNumpadDecimal}
	d, p, r := kp.keys(keys, keys, nil, true)
	if d != nil || p != nil || r != nil {
		t.Errorf("NumLock on: down %v pressed %v released %v, want none", d, p, r)
	}
}

func TestKeypadOtherKeys(t *testing.T) {
	// Keypad 5, + - * / and = are no key in either state; Enter is Enter in both.
	for _, numLock := range []bool{false, true} {
		var kp keypad
		keys := []ebiten.Key{ebiten.KeyNumpad5, ebiten.KeyNumpadAdd, ebiten.KeyNumpadSubtract,
			ebiten.KeyNumpadMultiply, ebiten.KeyNumpadDivide, ebiten.KeyNumpadEqual, ebiten.KeyNumpadEnter}
		_, p, _ := kp.keys(keys, keys, nil, numLock)
		if !reflect.DeepEqual(p, []render.Key{render.KeyEnter}) {
			t.Errorf("NumLock %v: pressed %v, want [Enter]", numLock, p)
		}
	}
}

func TestKeypadKeepsMeaningWhileHeld(t *testing.T) {
	var kp keypad
	four := []ebiten.Key{ebiten.KeyNumpad4}
	kp.keys(four, four, nil, false) // pressed with NumLock off: Left
	// NumLock goes on while it is held: still Left, and released as Left.
	if d, _, _ := kp.keys(four, nil, nil, true); !reflect.DeepEqual(d, []render.Key{render.KeyLeft}) {
		t.Errorf("held after toggle: %v, want [Left]", d)
	}
	if _, _, r := kp.keys(nil, nil, four, true); !reflect.DeepEqual(r, []render.Key{render.KeyLeft}) {
		t.Errorf("released after toggle as %v, want [Left]", r)
	}
	// The next press takes the new state: NumLock on, no key.
	if d, p, _ := kp.keys(four, four, nil, true); d != nil || p != nil {
		t.Errorf("pressed again with NumLock on: down %v pressed %v, want none", d, p)
	}
	// Pressed with NumLock on and released after it went off: no key either way.
	if d, _, r := kp.keys(nil, nil, four, false); d != nil || r != nil {
		t.Errorf("released after toggling off: down %v released %v, want none", d, r)
	}
}

func TestKeypadPressAndReleaseInOneFrame(t *testing.T) {
	var kp keypad
	one := []ebiten.Key{ebiten.KeyNumpad1}
	_, p, r := kp.keys(nil, one, one, false)
	if !reflect.DeepEqual(p, []render.Key{render.KeyEnd}) || !reflect.DeepEqual(r, []render.Key{render.KeyEnd}) {
		t.Errorf("tap: pressed %v released %v, want [End] both", p, r)
	}
	if len(kp.held) != 0 {
		t.Errorf("held after the tap: %v", kp.held)
	}
}

func TestKeypadForgetsKeysNoLongerDown(t *testing.T) {
	// Focus lost while a key is held: no release comes, and the key leaves the
	// down list. It must not keep its old meaning for the next press.
	var kp keypad
	nine := []ebiten.Key{ebiten.KeyNumpad9}
	kp.keys(nine, nine, nil, false)
	kp.keys(nil, nil, nil, false)
	if len(kp.held) != 0 {
		t.Fatalf("held after the key left: %v", kp.held)
	}
	if _, p, _ := kp.keys(nine, nine, nil, true); p != nil {
		t.Errorf("new press with NumLock on: %v, want none", p)
	}
}

func TestKeypadLeavesOtherKeysAlone(t *testing.T) {
	var kp keypad
	keys := []ebiten.Key{ebiten.KeyA, ebiten.KeyHome, ebiten.KeyShiftLeft, ebiten.KeyNumpad8}
	d, _, _ := kp.keys(keys, nil, nil, false)
	if want := []render.Key{render.KeyA, render.KeyHome, render.KeyUp}; !reflect.DeepEqual(d, want) {
		t.Errorf("down %v, want %v", d, want)
	}
}
