package ebitenbackend

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpfaulkner/guie/render"
)

func TestMapKeysTranslates(t *testing.T) {
	in := []ebiten.Key{ebiten.KeyEnter, ebiten.KeyA, ebiten.KeyDigit5, ebiten.KeyArrowLeft}
	want := []render.Key{render.KeyEnter, render.KeyA, render.Key5, render.KeyLeft}

	got := mapKeys(in)
	if len(got) != len(want) {
		t.Fatalf("mapKeys: got %d keys %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mapKeys[%d]: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestMapKeysDropsUnmapped(t *testing.T) {
	// Modifier keys are not in keyMap; they must be dropped, leaving only the
	// keys that have a backend-neutral equivalent.
	in := []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyB, ebiten.KeyControlLeft}
	got := mapKeys(in)
	if len(got) != 1 || got[0] != render.KeyB {
		t.Errorf("mapKeys with modifiers: got %v, want [KeyB]", got)
	}
}

func TestMapKeysEmpty(t *testing.T) {
	if got := mapKeys(nil); len(got) != 0 {
		t.Errorf("mapKeys(nil): got %v, want empty", got)
	}
}

func TestMapKeysFunctionEditingAndPunctuation(t *testing.T) {
	in := []ebiten.Key{
		ebiten.KeyF1, ebiten.KeyF12, ebiten.KeyInsert,
		ebiten.KeyMinus, ebiten.KeyEqual, ebiten.KeyBracketLeft, ebiten.KeyBracketRight,
		ebiten.KeyBackslash, ebiten.KeySemicolon, ebiten.KeyQuote, ebiten.KeyComma,
		ebiten.KeyPeriod, ebiten.KeySlash, ebiten.KeyBackquote,
	}
	want := []render.Key{
		render.KeyF1, render.KeyF12, render.KeyInsert,
		render.KeyMinus, render.KeyEqual, render.KeyBracketLeft, render.KeyBracketRight,
		render.KeyBackslash, render.KeySemicolon, render.KeyQuote, render.KeyComma,
		render.KeyPeriod, render.KeySlash, render.KeyBackquote,
	}
	got := mapKeys(in)
	if len(got) != len(want) {
		t.Fatalf("mapKeys: got %d keys %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mapKeys[%d]: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestF1ToF12AreConsecutive(t *testing.T) {
	// Callers index F-keys by offset from KeyF1, so the block must stay in order.
	for i := 0; i < 12; i++ {
		if got := render.KeyF1 + render.Key(i); keyMap[ebiten.KeyF1+ebiten.Key(i)] != got {
			t.Errorf("F%d does not map to KeyF1+%d", i+1, i)
		}
	}
}

func TestKeypadEnterIsEnter(t *testing.T) {
	got := mapKeys([]ebiten.Key{ebiten.KeyNumpadEnter})
	if len(got) != 1 || got[0] != render.KeyEnter {
		t.Errorf("keypad Enter mapped to %v, want KeyEnter", got)
	}
}
