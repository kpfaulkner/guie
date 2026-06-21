// Command layouts demonstrates the layout engine: nested containers using
// VBox / HBox / Grid / Stack, per-child weights and alignment, padding and
// themed panel colours. Resize the window and everything reflows.
//
// Run with: go run ./examples/layouts
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/kpfaulkner/guie/geom"
	"github.com/kpfaulkner/guie/ui"
)

// panel is a coloured box with a single centered label, built from a Stack.
func panel(bg color.Color, label string, txt color.Color) *ui.Container {
	c := ui.NewContainer()
	c.SetBackground(bg)
	c.SetLayout(ui.NewStack())
	c.Add(ui.NewLabel(label, ui.LabelColour(txt)), ui.Align(geom.AlignCenter))
	return c
}

func main() {
	app := ui.NewApp(
		ui.WithTitle("guie — layouts"),
		ui.WithSize(720, 480),
	)
	pal := app.Theme().Palette

	root := ui.NewContainer()
	root.SetLayout(ui.VBox(12))
	root.SetPadding(geom.UniformInsets(16))

	// A natural-height title that stretches across the width.
	root.Add(ui.NewLabel("Layouts: VBox > [ title, weighted HBox, Grid ]"))

	// A weighted row: the second panel is twice as wide as the first.
	row := ui.NewContainer()
	row.SetLayout(ui.HBox(12))
	row.Add(panel(pal.Surface, "HBox weight 1", pal.Text), ui.Weight(1))
	row.Add(panel(pal.Primary, "HBox weight 2", pal.OnPrimary), ui.Weight(2))
	root.Add(row, ui.Weight(2))

	// A 3-column grid that fills the remaining height. The first panel spans all
	// three columns as a header; a later one spans two columns.
	grid := ui.NewContainer()
	grid.SetLayout(ui.NewGrid(3, 8))
	grid.Add(panel(pal.Primary, "header (spans 3 columns)", pal.OnPrimary), ui.Span(3, 1))
	colours := []color.Color{
		color.RGBA{0x8a, 0x4a, 0x4a, 0xff}, color.RGBA{0x4a, 0x8a, 0x5a, 0xff}, color.RGBA{0x4a, 0x5a, 0x8a, 0xff},
		color.RGBA{0x8a, 0x7a, 0x4a, 0xff},
	}
	for i, cc := range colours {
		grid.Add(panel(cc, fmt.Sprintf("cell %d", i+1), pal.Text))
	}
	grid.Add(panel(color.RGBA{0x4a, 0x8a, 0x8a, 0xff}, "wide (spans 2)", pal.Text), ui.Span(2, 1))
	root.Add(grid, ui.Weight(3))

	app.SetContent(root)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
