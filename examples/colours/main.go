// Command colours demonstrates per-widget colour control. Each widget resolves
// its colours through theme "roles" (ui.RolePrimary, ui.RoleText, ...). Any role
// can be overridden per widget with SetColour and read back (effective colour)
// with ColourOf. The swatch buttons recolour the sample button's RolePrimary at
// runtime; "Reset" clears the override so it falls back to the theme. A second
// sample button is left untouched to show overrides are per-widget.
//
// Run with: go run ./examples/colours
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — colours"),
		ui.WithSize(560, 360),
	)

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(14))
	root.SetPadding(geom.UniformInsets(20))

	root.Add(ui.NewLabel("Per-widget colour overrides (SetColour / ColourOf):"))

	// The sample whose fill (RolePrimary) we recolour, plus an untouched one.
	sample := ui.NewButton("Sample")
	other := ui.NewButton("Untouched")
	samples := ui.NewContainer()
	samples.SetLayout(ui.HBox(10))
	samples.Add(sample)
	samples.Add(other)
	root.Add(samples, ui.Align(geom.AlignStart))

	status := ui.NewLabel("")
	update := func() {
		status.SetText("sample RolePrimary = " + hex(sample.ColourOf(ui.RolePrimary)))
	}
	update()
	root.Add(status)

	// Swatches recolour the sample's RolePrimary; Reset clears the override.
	swatch := func(label string, c color.Color) *ui.Button {
		b := ui.NewButton(label)
		b.OnClick(func() {
			sample.SetColour(ui.RolePrimary, c) // nil clears the override
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

	// Per-widget text-colour overrides (RoleText) on labels.
	a := ui.NewLabel("Accent-coloured text")
	a.SetColour(ui.RoleText, color.RGBA{0x5D, 0x86, 0xC4, 0xff})
	root.Add(a)

	b := ui.NewLabel("Muted text")
	b.SetColour(ui.RoleText, color.RGBA{0x8A, 0x8A, 0x99, 0xff})
	root.Add(b, ui.Weight(1))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
