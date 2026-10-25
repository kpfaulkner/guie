package ui

import (
	"github.com/kpfaulkner/guie/theme"
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
)

func TestMixColourEndpointsAndMidpoint(t *testing.T) {
	black, white := color.RGBA{A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if mixColour(black, white, 0) != color.Color(black) {
		t.Error("t=0 should give a")
	}
	if mixColour(black, white, 1) != color.Color(white) {
		t.Error("t=1 should give b")
	}
	r, g, b, a := mixColour(black, white, 0.5).RGBA()
	if r>>8 < 126 || r>>8 > 128 || r != g || g != b || a>>8 != 255 {
		t.Errorf("midpoint: got %d %d %d %d", r>>8, g>>8, b>>8, a>>8)
	}
}

func TestFadeSnapsBeforeMount(t *testing.T) {
	b := NewButton("x")
	b.HandleEvent(&Event{Type: EventPointerEnter})
	if b.hoverFade.level != 1 {
		t.Errorf("unmounted fade should snap, got %v", b.hoverFade.level)
	}
}

func TestButtonHoverFadesInOverFrames(t *testing.T) {
	a, _, _ := tooltipApp(t)
	b := a.root.(*Button)
	moveTo(t, a, geom.Point{X: 20, Y: 20})
	if !b.hover {
		t.Fatal("button should be hovered")
	}
	if l := b.hoverFade.level; l < 0 || l >= 1 {
		t.Errorf("fade should be in progress right after hover, got %v", l)
	}
	for range fadeFrames {
		moveTo(t, a, geom.Point{X: 20, Y: 20})
	}
	if b.hoverFade.level != 1 {
		t.Errorf("fade should land on 1, got %v", b.hoverFade.level)
	}
	moveTo(t, a, geom.Point{X: -5, Y: -5})
	for range fadeFrames {
		moveTo(t, a, geom.Point{X: -5, Y: -5})
	}
	if b.hoverFade.level != 0 {
		t.Errorf("fade should return to 0 on leave, got %v", b.hoverFade.level)
	}
}

// fadeFrames is enough frames for any fade to have landed.
const fadeFrames = 12

func TestTintMovesAwayFromThePage(t *testing.T) {
	lum := func(c color.Color) uint32 {
		r, g, b, _ := c.RGBA()
		return (299*r + 587*g + 114*b) / 1000
	}
	dark := color.RGBA{R: 0x26, G: 0x26, B: 0x32, A: 255}
	if lum(tint(dark, theme.DefaultStates().Hover)) <= lum(dark) {
		t.Error("tinting a dark surface should lighten it")
	}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if lum(tint(white, theme.DefaultStates().Hover)) >= lum(white) {
		t.Error("tinting a white surface should darken it (lighten would be a no-op)")
	}
}

func TestTableStripedToggles(t *testing.T) {
	tb := NewTable([]Column{{Title: "A"}})
	if tb.striped {
		t.Error("striping should be off by default")
	}
	tb.SetStriped(true)
	if !tb.striped {
		t.Error("SetStriped(true) did not stick")
	}
}
