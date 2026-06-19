// Command colors demonstrates per-widget color control. Each widget resolves
// its colors through theme "roles" (ui.RolePrimary, ui.RoleText, ...). Any role
// can be overridden per widget with SetColor and read back (effective color)
// with ColorOf. The swatch buttons recolor the sample button's RolePrimary at
// runtime; "Reset" clears the override so it falls back to the theme. A second
// sample button is left untouched to show overrides are per-widget.
//
// Run with: go run ./examples/colors
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/uiframework/geom"
	"github.com/kpfaulkner/uiframework/ui"
)

func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("uiframework — colors"),
		ui.WithSize(560, 360),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Per-widget color overrides (SetColor / ColorOf):"))

	// The sample whose fill (RolePrimary) we recolor, plus an untouched one.
	sample := ui.NewButton("Sample")
	other := ui.NewButton("Untouched")
	samples := ui.NewContainer()
	samples.SetLayout(ui.HBox(10))
	samples.Add(sample)
	samples.Add(other)
	root.Add(samples, ui.Align(geom.AlignStart))

	status := ui.NewLabel("")
	update := func() {
		status.SetText("sample RolePrimary = " + hex(sample.ColorOf(ui.RolePrimary)))
	}
	update()
	root.Add(status)

	// Swatches recolor the sample's RolePrimary; Reset clears the override.
	swatch := func(label string, c color.Color) *ui.Button {
		b := ui.NewButton(label)
		b.OnClick(func() {
			sample.SetColor(ui.RolePrimary, c) // nil clears the override
			update()
		})
		return b
	}
	swatches := ui.NewContainer()
	swatches.SetLayout(ui.HBox(10))
	swatches.Add(swatch("Crimson", color.RGBA{0xC0, 0x39, 0x2B, 0xff}))
	swatches.Add(swatch("Forest", color.RGBA{0x2E, 0x7D, 0x32, 0xff}))
	swatches.Add(swatch("Royal", color.RGBA{0x3B, 0x5B, 0xDB, 0xff}))
	swatches.Add(swatch("Reset", nil))
	root.Add(swatches, ui.Align(geom.AlignStart))

	// Per-widget text-color overrides (RoleText) on labels.
	a := ui.NewLabel("Accent-colored text")
	a.SetColor(ui.RoleText, color.RGBA{0x5D, 0x86, 0xC4, 0xff})
	root.Add(a)

	b := ui.NewLabel("Muted text")
	b.SetColor(ui.RoleText, color.RGBA{0x8A, 0x8A, 0x99, 0xff})
	root.Add(b, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
