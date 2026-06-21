package ui

import (
	"image/color"
	"testing"
)

func sameColour(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func TestColourOfFallsBackToTheme(t *testing.T) {
	app := NewApp()
	l := NewLabel("x")
	app.SetContent(l)
	pal := app.Theme().Palette

	if !sameColour(l.ColourOf(RoleSurface), pal.Surface) {
		t.Fatal("unset role should return the theme palette colour")
	}
	if !sameColour(l.ColourOf(RoleText), pal.Text) {
		t.Fatal("unset RoleText should be the theme text colour")
	}
}

func TestSetColourOverridesEffective(t *testing.T) {
	app := NewApp()
	l := NewLabel("x")
	app.SetContent(l)
	pal := app.Theme().Palette

	red := color.RGBA{R: 0xff, A: 0xff}
	l.SetColour(RoleText, red)
	if !sameColour(l.ColourOf(RoleText), red) {
		t.Fatal("ColourOf should return the override")
	}
	// Other roles still fall back to the theme.
	if !sameColour(l.ColourOf(RoleBorder), pal.Border) {
		t.Fatal("non-overridden roles should still use the theme")
	}
}

func TestSetColourNilClearsOverride(t *testing.T) {
	app := NewApp()
	l := NewLabel("x")
	app.SetContent(l)
	pal := app.Theme().Palette

	l.SetColour(RoleText, color.RGBA{R: 0xff, A: 0xff})
	l.SetColour(RoleText, nil) // clear
	if !sameColour(l.ColourOf(RoleText), pal.Text) {
		t.Fatal("clearing an override should fall back to the theme")
	}
}

func TestButtonColourOptionMapsToRole(t *testing.T) {
	c := color.RGBA{R: 1, G: 2, B: 3, A: 0xff}
	tc := color.RGBA{R: 4, G: 5, B: 6, A: 0xff}
	b := NewButton("ok", ButtonColour(c), ButtonTextColour(tc))
	if !sameColour(b.ColourOf(RolePrimary), c) {
		t.Fatal("ButtonColour should set RolePrimary")
	}
	if !sameColour(b.ColourOf(RoleOnPrimary), tc) {
		t.Fatal("ButtonTextColour should set RoleOnPrimary")
	}
}

func TestContainerColourGetters(t *testing.T) {
	c := NewContainer()
	if c.Background() != nil {
		t.Fatal("a new container should be transparent (nil background)")
	}
	bg := color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
	c.SetBackground(bg)
	if !sameColour(c.Background(), bg) {
		t.Fatal("Background() should return the set colour")
	}
	bc := color.RGBA{R: 0x40, A: 0xff}
	c.SetBorder(bc, 2)
	gotCol, gotW := c.BorderColour()
	if !sameColour(gotCol, bc) || gotW != 2 {
		t.Fatalf("BorderColour() should return the set border, got %v / %v", gotCol, gotW)
	}
}

func TestColourOverrideAffectsAnyWidget(t *testing.T) {
	// The override mechanism lives on BaseWidget, so it works uniformly — e.g.
	// a Slider can have its primary colour overridden and read back.
	app := NewApp()
	s := NewSlider()
	app.SetContent(s)
	green := color.RGBA{G: 0xff, A: 0xff}
	s.SetColour(RolePrimary, green)
	if !sameColour(s.ColourOf(RolePrimary), green) {
		t.Fatal("any widget should support colour overrides via BaseWidget")
	}
}
