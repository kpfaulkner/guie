package ui

import (
	"image/color"
	"testing"
)

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func TestColorOfFallsBackToTheme(t *testing.T) {
	app := NewApp()
	l := NewLabel("x")
	app.SetContent(l)
	pal := app.Theme().Palette

	if !sameColor(l.ColorOf(RoleSurface), pal.Surface) {
		t.Fatal("unset role should return the theme palette color")
	}
	if !sameColor(l.ColorOf(RoleText), pal.Text) {
		t.Fatal("unset RoleText should be the theme text color")
	}
}

func TestSetColorOverridesEffective(t *testing.T) {
	app := NewApp()
	l := NewLabel("x")
	app.SetContent(l)
	pal := app.Theme().Palette

	red := color.RGBA{R: 0xff, A: 0xff}
	l.SetColor(RoleText, red)
	if !sameColor(l.ColorOf(RoleText), red) {
		t.Fatal("ColorOf should return the override")
	}
	// Other roles still fall back to the theme.
	if !sameColor(l.ColorOf(RoleBorder), pal.Border) {
		t.Fatal("non-overridden roles should still use the theme")
	}
}

func TestSetColorNilClearsOverride(t *testing.T) {
	app := NewApp()
	l := NewLabel("x")
	app.SetContent(l)
	pal := app.Theme().Palette

	l.SetColor(RoleText, color.RGBA{R: 0xff, A: 0xff})
	l.SetColor(RoleText, nil) // clear
	if !sameColor(l.ColorOf(RoleText), pal.Text) {
		t.Fatal("clearing an override should fall back to the theme")
	}
}

func TestButtonColorOptionMapsToRole(t *testing.T) {
	c := color.RGBA{R: 1, G: 2, B: 3, A: 0xff}
	tc := color.RGBA{R: 4, G: 5, B: 6, A: 0xff}
	b := NewButton("ok", ButtonColor(c), ButtonTextColor(tc))
	if !sameColor(b.ColorOf(RolePrimary), c) {
		t.Fatal("ButtonColor should set RolePrimary")
	}
	if !sameColor(b.ColorOf(RoleOnPrimary), tc) {
		t.Fatal("ButtonTextColor should set RoleOnPrimary")
	}
}

func TestContainerColorGetters(t *testing.T) {
	c := NewContainer()
	if c.Background() != nil {
		t.Fatal("a new container should be transparent (nil background)")
	}
	bg := color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
	c.SetBackground(bg)
	if !sameColor(c.Background(), bg) {
		t.Fatal("Background() should return the set color")
	}
	bc := color.RGBA{R: 0x40, A: 0xff}
	c.SetBorder(bc, 2)
	gotCol, gotW := c.BorderColor()
	if !sameColor(gotCol, bc) || gotW != 2 {
		t.Fatalf("BorderColor() should return the set border, got %v / %v", gotCol, gotW)
	}
}

func TestColorOverrideAffectsAnyWidget(t *testing.T) {
	// The override mechanism lives on BaseWidget, so it works uniformly — e.g.
	// a Slider can have its primary color overridden and read back.
	app := NewApp()
	s := NewSlider()
	app.SetContent(s)
	green := color.RGBA{G: 0xff, A: 0xff}
	s.SetColor(RolePrimary, green)
	if !sameColor(s.ColorOf(RolePrimary), green) {
		t.Fatal("any widget should support color overrides via BaseWidget")
	}
}
