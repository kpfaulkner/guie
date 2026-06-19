package ui_test

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func sameColor(a, b color.Color) bool {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

// These tests exercise the package strictly through its exported API, the way a
// consumer would use it.

func TestButtonOnClick(t *testing.T) {
	clicked := false
	b := ui.NewButton("ok")
	b.OnClick(func() { clicked = true })

	b.HandleEvent(&ui.Event{Type: ui.EventClick})
	if !clicked {
		t.Fatal("OnClick handler should fire on EventClick")
	}
}

func TestCheckboxOnChange(t *testing.T) {
	var got bool
	fired := false
	c := ui.NewCheckbox("enable")
	c.OnChange(func(v bool) { got = v; fired = true })

	c.HandleEvent(&ui.Event{Type: ui.EventClick})
	if !c.IsChecked() || !fired || got != true {
		t.Fatalf("clicking should check the box and fire OnChange(true): checked=%v fired=%v got=%v", c.IsChecked(), fired, got)
	}
}

func TestColorOverrideAndFallback(t *testing.T) {
	app := ui.NewApp()
	lbl := ui.NewLabel("x")
	app.SetContent(lbl)

	// Unset role falls back to the theme.
	if !sameColor(lbl.ColorOf(ui.RoleSurface), app.Theme().Palette.Surface) {
		t.Fatal("unset role should return the theme color")
	}
	// Override is returned as the effective color.
	red := color.RGBA{R: 0xff, A: 0xff}
	lbl.SetColor(ui.RoleText, red)
	if !sameColor(lbl.ColorOf(ui.RoleText), red) {
		t.Fatal("ColorOf should return the override")
	}
}

func TestListSelectionViaPublicAPI(t *testing.T) {
	app := ui.NewApp()
	list := ui.NewList([]string{"a", "b", "c"})
	app.SetContent(list) // mounts it, so the theme font is available
	rh := list.RowHeight()
	list.SetBounds(geom.Rect{X: 0, Y: 0, W: 120, H: rh * 3})

	sel := -1
	list.OnSelect(func(i int) { sel = i })
	list.HandleEvent(&ui.Event{Type: ui.EventClick, Pos: geom.Point{X: 5, Y: rh*1 + rh/2}})

	if list.Selected() != 1 || sel != 1 {
		t.Fatalf("clicking row 1 should select it: selected=%d cb=%d", list.Selected(), sel)
	}
}

// ExampleApp shows the typical shape of a guie program: construct an App,
// build a widget tree, wire callbacks, set the content and run.
func ExampleApp() {
	app := ui.NewApp(ui.WithTitle("Demo"), ui.WithSize(320, 240))

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(8))
	root.SetPadding(geom.UniformInsets(12))

	status := ui.NewLabel("clicks: 0")
	clicks := 0
	btn := ui.NewButton("Click me")
	btn.OnClick(func() {
		clicks++
		status.SetText(fmt.Sprintf("clicks: %d", clicks))
	})

	root.Add(btn)
	root.Add(status)
	app.SetContent(root)

	// app.Run() // starts the main loop; omitted so the example stays headless.
}

// ExampleLabel_SetColor shows overriding a widget's color by role.
func ExampleLabel_SetColor() {
	label := ui.NewLabel("warning")
	label.SetColor(ui.RoleText, color.RGBA{R: 0xff, G: 0x55, B: 0x55, A: 0xff})
}
