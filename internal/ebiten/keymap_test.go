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
